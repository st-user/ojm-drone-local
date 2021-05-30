package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
)

var ENV = loadEnv()
var routineCoordinator = RoutineCoordinator{}
var Log = NewLogger(ENV["LOG_LEVEL"])

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
	url := ENV["SIGNALING_ENDPOINT"] + "/generateKey"
	secret := ENV["SECRET"]

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
	log.Println("End waiting for the waitgroup to be done.")

	routineCoordinator.InitRoutineCoordinator(false)
	decoder := json.NewDecoder(r.Body)
	bodyJson := make(map[string]string)
	err := decoder.Decode(&bodyJson)

	if err != nil {
		return nil, err
	}

	url := ENV["SIGNALING_ENDPOINT"] + "/signaling?startKey=" + bodyJson["startKey"]
	url = strings.ReplaceAll(url, "http://", "ws://")
	url = strings.ReplaceAll(url, "https://", "wss://")

	conn, _, err := websocket.DefaultDialer.Dial(url, nil)

	if err != nil {
		return nil, err
	}

	go startSignalingConnection(conn)

	responseBody := map[string]interface{}{}
	return &responseBody, nil
}

func startSignalingConnection(connection *websocket.Conn) {
	defer connection.Close()

	var rtcHandler RTCHandler

	for {
		select {
		case <-routineCoordinator.StopSignalChannel:
			log.Println("Stop Signaling channel.")
			return
		default:

			_, message, err := connection.ReadMessage()
			if err != nil {
				log.Println(err)
				continue
			}

			rtcMessageData, err := NewRTCMessageData(&message)
			if err != nil {
				log.Println(err)
				continue
			}
			messageType := rtcMessageData.MessageType

			switch messageType {
			case "iceServerInfo":

				config, err := rtcMessageData.ToConfiguration()
				if err != nil {
					log.Println(err)
					continue
				}
				rtcHandler, err = NewRTCHandler(config)
				if err != nil {
					log.Println(err)
					continue
				}

			case "canOffer":
				log.Println("canOffer")

				peerType := rtcMessageData.ToPeerType()
				state := rtcHandler.DecidePeerState(peerType)

				connection.WriteJSON(map[string]interface{}{
					"messageType":      "canOffer",
					"peerConnectionId": peerType.PeerConnectionId,
					"state":            state,
				})

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
				log.Println("offer")

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
					log.Println(err)
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
					log.Println(err)
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

	port := ENV["PORT"]
	log.Println("PORT:" + port)

	routineCoordinator.InitRoutineCoordinator(true)
	routineCoordinator.IsStopped = true

	statics := NewStatics()
	server := NewOutboundRelayMessageServer()

	http.HandleFunc("/", index(&statics))
	http.HandleFunc("/main.js", statics.ToHandleFunc("main.js"))
	http.HandleFunc("/main.css", statics.ToHandleFunc("main.css"))
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
