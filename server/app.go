package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
	"github.com/st-user/ojm-drone-local/applog"
	"github.com/st-user/ojm-drone-local/appos"
	"github.com/st-user/ojm-drone-local/env"
)

var routineCoordinator = RoutineCoordinator{}
var applicationStates = NewApplicationStates()
var keyChainManager appos.KeyChainManager

func toEndpointUrlWithTrailingSlash() string {
	endpoint := env.Get("SIGNALING_ENDPOINT")
	if string(endpoint[len(endpoint)-1]) != "/" {
		endpoint = endpoint + "/"
	}
	return endpoint
}

func createAuthorizationRequest(path string, token string) (*http.Request, error) {

	url := toEndpointUrlWithTrailingSlash() + path

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "bearer "+token)

	return req, nil
}

func checkAccessTokenSaved(w http.ResponseWriter, r *http.Request) (*map[string]interface{}, error) {

	_, desc, err := keyChainManager.GetTokenAndDesc()

	if err != nil {
		return nil, err
	}

	return &map[string]interface{}{
		"accessTokenDesc": desc,
	}, nil
}

func updateAccessToken(w http.ResponseWriter, r *http.Request) (*map[string]interface{}, error) {

	decoder := json.NewDecoder(r.Body)
	bodyJson := make(map[string]string)
	err := decoder.Decode(&bodyJson)

	if err != nil {
		return nil, err
	}
	token := bodyJson["accessToken"]

	req, err := createAuthorizationRequest("validateAccessToken", token)

	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	res, err := client.Do(req)

	if err != nil || res.StatusCode != 200 {
		return nil, fmt.Errorf("encounters an error during handling response. %v %v", err, res.Status)
	}
	defer res.Body.Close()

	_, desc, err := keyChainManager.UpdateToken(token)

	if err != nil {
		return nil, err
	}

	return &map[string]interface{}{
		"accessTokenDesc": desc,
	}, nil
}

func deleteAccessToken(w http.ResponseWriter, r *http.Request) (*map[string]interface{}, error) {

	err := keyChainManager.DeleteToken()

	if err != nil {
		return nil, err
	}

	return &map[string]interface{}{}, nil
}

func checkDroneHealth(w http.ResponseWriter, r *http.Request) (*map[string]interface{}, error) {
	return &map[string]interface{}{
		"health":       applicationStates.DroneStates.DroneHealth(),
		"batteryLevel": applicationStates.DroneStates.BatteryLevel(),
	}, nil
}

func generateKey(w http.ResponseWriter, r *http.Request) (*map[string]interface{}, error) {

	token, err := keyChainManager.GetToken()

	if err != nil {
		return nil, err
	}

	req, err := createAuthorizationRequest("generateKey", token)

	if err != nil {
		return nil, err
	}

	client := &http.Client{}
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
	applog.Info("End waiting for the waitgroup to be done.")

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
		maxRetry := env.GetInt("SIGNALING_ENDPOINT_MAX_RETRY")
		if maxRetry < retryCount {
			applog.Info("Fails to connect to the signaling channel. Retry count exceeds max.")
			routineCoordinator.StopApp()
			return
		}

		interval := env.GetDuration("SIGNALING_ENDPOINT_RETRY_INTERVAL")
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
			applog.Info("Stop Signaling channel.")
			return
		default:

			_, message, err := connection.ReadMessage()
			if err != nil {
				consecutiveErrorOnReadCount++
				applog.Info("%v", err)
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
				applog.Info("%v", err)
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
					applog.Info("%v", err)
					continue
				}

				err = rtcHandler.SetConfig(config)
				if err != nil {
					applog.Info("%v", err)
					continue
				}

			case "canOffer":
				applog.Info("canOffer")

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
						applog.Info("Primary peer is requesting new connection. Restart the application.")
						routineCoordinator.StopApp()
						write()
						return
					} else {
						applog.Info("Audience peer(%v) is requesting new connectiond.", peerType.PeerConnectionId)
						rtcHandler.SendAudienceRTCStopChannel(peerType.PeerConnectionId)
					}
				}
				write()

			case "close":

				applog.Info("One of the peers has been closed.")
				peerType := rtcMessageData.ToPeerType()
				if rtcHandler.IsPrimary(peerType.PeerConnectionId) {
					applog.Info("Primary peer has been closed. Restart the application.")
					routineCoordinator.StopApp()
					return

				} else {
					if !peerType.IsPrimary {
						applog.Info("Audience peer has been closed. %v", peerType.PeerConnectionId)
						rtcHandler.SendAudienceRTCStopChannel(peerType.PeerConnectionId)
					}
				}

			case "offer":
				applog.Info("offer")

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
					applog.Info("%v", err)
					writeErrAnswer()
					continue
				}

				var localDescription *webrtc.SessionDescription

				if rtcHandler.IsPrimary(peerConnectionId) {
					drone := NewDrone()
					drone.Start(&routineCoordinator, applicationStates)
					localDescription, err = rtcHandler.StartPrimaryConnection(sdp, &routineCoordinator)
				} else {
					localDescription, err = rtcHandler.StartAudienceConnection(peerConnectionId, sdp, &routineCoordinator)
				}

				if err != nil {
					applog.Info("%v", err)
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

	port := env.Get("PORT")
	applog.Info("PORT:" + port)

	routineCoordinator.InitRoutineCoordinator(true)
	routineCoordinator.IsStopped = true

	km, err := appos.NewKeyChainManager()
	if err != nil {
		panic(err)
	}
	keyChainManager = km

	statics := NewStatics()
	server := NewOutboundRelayMessageServer()

	HandleFuncJSON("/checkAccessTokenSaved", checkAccessTokenSaved)
	HandleFuncJSON("/updateAccessToken", updateAccessToken)
	HandleFuncJSON("/deleteAccessToken", deleteAccessToken)
	HandleFuncJSON("/checkDroneHealth", checkDroneHealth)
	HandleFuncJSON("/generateKey", generateKey)
	HandleFuncJSON("/startApp", startApp)
	HandleFuncJSON("/healthCheck", healthCheck)
	HandleFuncJSON("/takeoff", takeoff)
	HandleFuncJSON("/land", land)

	http.HandleFunc("/state", state(&server))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		filename := r.URL.Path[len("/"):]

		if filename == "index.html" || filename == "" {
			if !routineCoordinator.IsStopped {
				routineCoordinator.StopApp()
			}
		}

		statics.HandleStatic(w, r)
	})

	log.Fatal(http.ListenAndServe("localhost:"+port, nil))
}

func main() {
	go routes()
	go appos.OpenBrowser("http://localhost:"+env.Get("PORT"), 3*time.Second)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)

	<-signalChan
	fmt.Println("Are you sure you want to stop the application? If so, press 'ctrl+c' twice.")
	<-signalChan
	<-signalChan
	os.Exit(2)
}
