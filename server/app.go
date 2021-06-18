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

func checkApplicationStates(w http.ResponseWriter, r *http.Request) (*map[string]interface{}, error) {

	_, desc, err := keyChainManager.GetTokenAndDesc()

	if err != nil {
		return nil, err
	}

	return &map[string]interface{}{
		"accessTokenDesc":  desc,
		"applicationState": applicationStates.GetState(),
		"startKey":         applicationStates.GetStartKey(),
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
		return nil, fmt.Errorf("encounters an error during handling response. %v", err)
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

func stopApp(w http.ResponseWriter, r *http.Request) (*map[string]interface{}, error) {
	applicationStates.StartStopMux.Lock()
	defer applicationStates.StartStopMux.Unlock()

	applicationStates.SetState(APPLICATION_STATE_INIT)
	applicationStates.SetStartKey("")
	if !routineCoordinator.IsStopped {
		routineCoordinator.StopApp()
	}

	responseBody := map[string]interface{}{}
	return &responseBody, nil
}

func startApp(w http.ResponseWriter, r *http.Request) (*map[string]interface{}, error) {
	applicationStates.StartStopMux.Lock()
	defer applicationStates.StartStopMux.Unlock()

	if applicationStates.IsStarted() {
		applog.Info("Application has already been started.")
		responseBody := map[string]interface{}{}
		return &responseBody, nil
	}
	applicationStates.Start()

	decoder := json.NewDecoder(r.Body)
	bodyJson := make(map[string]string)
	err := decoder.Decode(&bodyJson)

	if err != nil {
		return nil, err
	}

	startKey := bodyJson["startKey"]
	err = startAppFrom(startKey)

	if err != nil {
		return nil, err
	}

	responseBody := map[string]interface{}{}
	return &responseBody, nil
}

func startAppFrom(startKey string) error {
	routineCoordinator.WaitUntilReleasingSocket()
	applog.Info("End waiting for the waitgroup to be done.")

	routineCoordinator.InitRoutineCoordinator(false)

	startKeyJson := map[string]string{
		"startKey": startKey,
	}
	startKeyJsonBytes, err := json.Marshal(startKeyJson)
	if err != nil {
		return err
	}

	rtcHandler := NewRTCHandler()
	err = negotiateSignalingConnection(startKeyJsonBytes, rtcHandler)
	if err != nil {
		return err
	}

	applicationStates.SetStartKey(startKey)

	return nil
}

func restartApp() {
	applicationStates.StartStopMux.Lock()
	defer applicationStates.StartStopMux.Unlock()

	exsitingStartKey := applicationStates.GetStartKey()
	if exsitingStartKey == "" {
		return
	}

	routineCoordinator.StopApp()

	err := startAppFrom(exsitingStartKey)

	if err != nil {
		applog.Warn("Failed to restart signaling connection. %v", err.Error())
	}
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
			restartApp()
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
						restartApp()
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
					restartApp()
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
					localDescription, err = rtcHandler.StartPrimaryConnection(sdp, &routineCoordinator, applicationStates)
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

func state(w http.ResponseWriter, r *http.Request) {

	server := NewApplicationStatesServer()
	server.Start(w, r, applicationStates)
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

func terminate(w http.ResponseWriter, r *http.Request) (*map[string]interface{}, error) {
	stopApp(w, r)

	go func() {
		applog.Warn("Application terminates...")
		time.Sleep(5 * time.Second)
		os.Exit(2)
	}()

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

	statics := NewStatics(applicationStates.SessionKey)

	HandleFuncJSON("/checkApplicationStates", checkApplicationStates)
	HandleFuncJSON("/updateAccessToken", updateAccessToken)
	HandleFuncJSON("/deleteAccessToken", deleteAccessToken)
	HandleFuncJSON("/generateKey", generateKey)
	HandleFuncJSON("/stopApp", stopApp)
	HandleFuncJSON("/startApp", startApp)
	HandleFuncJSON("/takeoff", takeoff)
	HandleFuncJSON("/land", land)
	HandleFuncJSON("/terminate", terminate)

	http.HandleFunc("/state", state)
	http.HandleFunc("/", statics.HandleStatic)

	log.Fatal(http.ListenAndServe("localhost:"+port, nil))
}

func main() {
	go routes()
	go func() {
		if env.GetBool("OPEN_BROWSER_ON_START_UP") {
			appos.OpenBrowser("http://localhost:"+env.Get("PORT"), 3*time.Second)
		}
	}()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)

	<-signalChan
	fmt.Println("Are you sure you want to stop the application? If so, press 'ctrl+c' twice.")
	<-signalChan
	<-signalChan
	os.Exit(2)
}
