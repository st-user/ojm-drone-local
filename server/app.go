package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
)

var ENV = loadEnv()
var routineCoordinator = RoutineCoordinator{}
var Log = NewLogger(ENV.Get("LOG_LEVEL"), ENV.Get("DUMP_LOG_FILE_PATH"))

func toEndpointUrlWithTrailingSlash() string {
	endpoint := ENV.Get("SIGNALING_ENDPOINT")
	if "/" != string(endpoint[len(endpoint)-1]) {
		endpoint = endpoint + "/"
	}
	return endpoint
}

func index(statics *Statics) func(w http.ResponseWriter, r *http.Request) {

	return func(w http.ResponseWriter, r *http.Request) {
		if !routineCoordinator.IsStopped {
			routineCoordinator.StopApp()
		}
		statics.ToHandleFunc("index.html")(w, r)
	}
}

func generateKey(w http.ResponseWriter, r *http.Request) (*map[string]interface{}, error) {

	client := &http.Client{}
	url := toEndpointUrlWithTrailingSlash() + "generateKey"
	secret := ENV.Get("SECRET")

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "bearer "+secret)

	res, err := client.Do(req)
	if err != nil || res.StatusCode != 200 {
		return nil, fmt.Errorf("encounters an error during handling response. %v %v", err, res.Status)
	}

	defer res.Body.Close()

	var result map[string]string
	body, err := io.ReadAll(res.Body)

	if err != nil {
		return nil, err
	}

	json.Unmarshal(body, &result)

	responseBody := map[string]interface{}{
		"startKey": result["startKey"],
	}
	return &responseBody, nil
}

func startApp(w http.ResponseWriter, r *http.Request) (*map[string]interface{}, error) {

	routineCoordinator.WaitUntilReleasingSocket()
	Log.Info("End waiting for the waitgroup to be done.")

	routineCoordinator.InitRoutineCoordinator(false)
	decoder := json.NewDecoder(r.Body)
	bodyJson := make(map[string]string)
	err := decoder.Decode(&bodyJson)

	if err != nil {
		return nil, err
	}

	startKeyJson := map[string]string{
		"startKey": bodyJson["startKey"],
	}
	startKeyJsonBytes, err := json.Marshal(startKeyJson)
	if err != nil {
		return nil, err
	}

	rtcHandler := NewRTCHandler()
	err = negotiateSignalingConnection(startKeyJsonBytes, rtcHandler)
	if err != nil {
		return nil, err
	}

	responseBody := map[string]interface{}{}
	return &responseBody, nil
}

func negotiateSignalingConnection(startKeyJsonBytes []byte, rtcHandler *RTCHandler) error {
	copyStartKeyJsonBytes := make([]byte, len(startKeyJsonBytes))
	copy(copyStartKeyJsonBytes, startKeyJsonBytes)

	baseUrl := toEndpointUrlWithTrailingSlash()
	ticketUrl := baseUrl + "ticket"

	res, err := http.Post(ticketUrl, "application/json", bytes.NewBuffer(startKeyJsonBytes))
	if err != nil || res.StatusCode != 200 {
		return fmt.Errorf("encounters an error during handling response. %v", err)
	}

	var ticketJson map[string]string
	json.NewDecoder(res.Body).Decode(&ticketJson)

	url := baseUrl + "signaling?ticket=" + ticketJson["ticket"]
	url = strings.ReplaceAll(url, "http://", "ws://")
	url = strings.ReplaceAll(url, "https://", "wss://")

	conn, _, err := websocket.DefaultDialer.Dial(url, nil)

	if err != nil {
		return err
	}
	var retryCount int

	go startSignalingConnection(conn, rtcHandler, func() {
		restartSignalingConnection(copyStartKeyJsonBytes, retryCount, rtcHandler)
	})

	return nil
}

func restartSignalingConnection(startKeyJsonBytes []byte, retryCount int, rtcHandler *RTCHandler) {
	b := make([]byte, len(startKeyJsonBytes))
	copy(b, startKeyJsonBytes)
	err := negotiateSignalingConnection(b, rtcHandler)
	if err != nil {
		maxRetry := ENV.GetInt("SIGNALING_ENDPOINT_MAX_RETRY")
		if maxRetry < retryCount {
			Log.Info("Fails to connect to the signaling channel. Retry count exceeds max.")
			routineCoordinator.StopApp()
			return
		}

		interval := ENV.GetDuration("SIGNALING_ENDPOINT_RETRY_INTERVAL")
		time.Sleep(interval)
		retryCount = retryCount + 1
		restartSignalingConnection(startKeyJsonBytes, retryCount, rtcHandler)
	}
}

func startSignalingConnection(connection *websocket.Conn, rtcHandler *RTCHandler, recoverFunc func()) {
	connectionStoppedChannel := make(chan struct{})

	go func() {
		select {
		case <-connectionStoppedChannel:
			connection.Close()
		case <-routineCoordinator.StopSignalChannel:
			connection.Close()
		}

	}()

	var consecutiveErrorOnReadCount int
	for {

		select {
		case <-routineCoordinator.StopSignalChannel:
			Log.Info("Stop Signaling channel.")
			return
		default:

			_, message, err := connection.ReadMessage()
			if err != nil {
				consecutiveErrorOnReadCount++
				Log.Info("%v", err)
				if 10 < consecutiveErrorOnReadCount {
					recoverFunc()
					close(connectionStoppedChannel)
					return
				}
				continue
			}
			consecutiveErrorOnReadCount = 0

			rtcMessageData, err := NewRTCMessageData(&message)
			if err != nil {
				Log.Info("%v", err)
				continue
			}
			messageType := rtcMessageData.MessageType

			switch messageType {
			case "ping":
				connection.WriteJSON(map[string]string{
					"messageType": "pong",
				})
			case "iceServerInfo":

				config, err := rtcMessageData.ToConfiguration()
				if err != nil {
					Log.Info("%v", err)
					continue
				}

				err = rtcHandler.SetConfig(config)
				if err != nil {
					Log.Info("%v", err)
					continue
				}

			case "canOffer":
				Log.Info("canOffer")

				peerType := rtcMessageData.ToPeerType()
				state := rtcHandler.DecidePeerState(peerType)

				write := func() {
					connection.WriteJSON(map[string]interface{}{
						"messageType":      "canOffer",
						"peerConnectionId": peerType.PeerConnectionId,
						"state":            state,
					})
				}

				if state == PEER_STATE_SAME {
					if peerType.IsPrimary {
						Log.Info("Primary peer is requesting new connection. Restart the application.")
						routineCoordinator.StopApp()
						write()
						return
					} else {
						Log.Info("Audience peer(%v) is requesting new connectiond.", peerType.PeerConnectionId)
						rtcHandler.SendAudienceRTCStopChannel(peerType.PeerConnectionId)
					}
				}
				write()

			case "close":

				Log.Info("One of the peers has been closed.")
				peerType := rtcMessageData.ToPeerType()
				if rtcHandler.IsPrimary(peerType.PeerConnectionId) {
					Log.Info("Primary peer has been closed. Restart the application.")
					routineCoordinator.StopApp()
					return

				} else {
					if !peerType.IsPrimary {
						Log.Info("Audience peer has been closed. %v", peerType.PeerConnectionId)
						rtcHandler.SendAudienceRTCStopChannel(peerType.PeerConnectionId)
					}
				}

			case "offer":
				Log.Info("offer")

				peerConnectionId := rtcMessageData.ToPeerConnectionId()

				writeErrAnswer := func() {
					rtcHandler.DeleteAudience(peerConnectionId)
					connection.WriteJSON(map[string]interface{}{
						"messageType":      "answer",
						"peerConnectionId": peerConnectionId,
						"err":              true,
					})
				}
				sdp, err := rtcMessageData.ToSessionDescription()
				if err != nil {
					Log.Info("%v", err)
					writeErrAnswer()
					continue
				}

				var localDescription *webrtc.SessionDescription

				if rtcHandler.IsPrimary(peerConnectionId) {
					drone := NewDrone()
					drone.Start(&routineCoordinator)
					localDescription, err = rtcHandler.StartPrimaryConnection(sdp, &routineCoordinator)
				} else {
					localDescription, err = rtcHandler.StartAudienceConnection(peerConnectionId, sdp, &routineCoordinator)
				}

				if err != nil {
					Log.Info("%v", err)
					writeErrAnswer()
					continue
				}

				connection.WriteJSON(map[string]interface{}{
					"messageType":      "answer",
					"peerConnectionId": peerConnectionId,
					"err":              false,
					"answer": map[string]string{
						"sdp":  localDescription.SDP,
						"type": localDescription.Type.String(),
					},
				})
			}

		}
	}
}

func healthCheck(w http.ResponseWriter, r *http.Request) (*map[string]interface{}, error) {
	responseBody := map[string]interface{}{}
	return &responseBody, nil
}

func state(server *OutboundRelayMessageServer) func(w http.ResponseWriter, r *http.Request) {

	return func(w http.ResponseWriter, r *http.Request) {
		server.HandleMessage(w, r, &routineCoordinator, func(text string) map[string]interface{} {
			result := map[string]interface{}{
				"messageType": "stateChange",
				"state":       text,
			}
			return result
		})
	}
}

func takeoff(w http.ResponseWriter, r *http.Request) (*map[string]interface{}, error) {

	routineCoordinator.SendDataChannelMessageChannel("takeoff")
	routineCoordinator.SendDroneCommandChannel(DroneCommand{
		CommandType: "takeoff",
	})

	responseBody := map[string]interface{}{}
	return &responseBody, nil
}

func land(w http.ResponseWriter, r *http.Request) (*map[string]interface{}, error) {

	routineCoordinator.SendDataChannelMessageChannel("land")
	routineCoordinator.SendDroneCommandChannel(DroneCommand{
		CommandType: "land",
	})

	responseBody := map[string]interface{}{}
	return &responseBody, nil
}

func routes() {

	port := ENV.Get("PORT")
	Log.Info("PORT:" + port)

	routineCoordinator.InitRoutineCoordinator(true)
	routineCoordinator.IsStopped = true

	statics := NewStatics()
	server := NewOutboundRelayMessageServer()

	http.HandleFunc("/", index(&statics))
	http.HandleFunc("/js/main.js", statics.ToHandleFunc("js/main.js"))
	http.HandleFunc("/js/style.js", statics.ToHandleFunc("js/style.js"))
	http.HandleFunc("/waiting.html", statics.ToHandleFunc("waiting.html"))

	HandleFuncJSON("/generateKey", generateKey)
	HandleFuncJSON("/startApp", startApp)
	HandleFuncJSON("/healthCheck", healthCheck)
	HandleFuncJSON("/takeoff", takeoff)
	HandleFuncJSON("/land", land)

	http.HandleFunc("/state", state(&server))

	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func main() {
	routes()
}
