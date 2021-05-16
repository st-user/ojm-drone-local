package main

import (
	"encoding/json"
	"log"
	"time"

	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
)

type RTCMessageData struct {
	MessageType string
	data        map[string]interface{}
}

type ICEServerInfo struct {
	Stun        string
	Turn        string
	Credentials ICEServerInfoCredential
}

type ICEServerInfoCredential struct {
	Username string
	Password string
}

func NewRTCMessageData(message *[]byte) (RTCMessageData, error) {
	messageJson := make(map[string]interface{})
	err := json.Unmarshal(*message, &messageJson)
	if err != nil {
		return RTCMessageData{}, err
	}
	messageType := messageJson["messageType"].(string)

	return RTCMessageData{
		MessageType: messageType,
		data:        messageJson,
	}, nil
}

func (d *RTCMessageData) ToConfiguration() (*webrtc.Configuration, error) {
	_iceServerInfo, exists := d.data["iceServerInfo"]
	config := webrtc.Configuration{}
	if exists {
		iceServerInfoMap := _iceServerInfo.(map[string]interface{})
		iceServerInfoMapBytes, err := json.Marshal(iceServerInfoMap)
		if err != nil {
			return &config, err
		}

		var iceServerInfo ICEServerInfo
		err = json.Unmarshal(iceServerInfoMapBytes, &iceServerInfo)
		if err != nil {
			return &config, err
		}

		config = webrtc.Configuration{
			ICEServers: []webrtc.ICEServer{
				{
					URLs: []string{iceServerInfo.Stun},
				},
				{
					URLs:       []string{iceServerInfo.Turn},
					Username:   iceServerInfo.Credentials.Username,
					Credential: iceServerInfo.Credentials.Password,
				},
			},
		}
	}
	return &config, nil
}

func (d *RTCMessageData) ToPeerConnectionId() float64 {
	return d.data["peerConnectionId"].(float64)
}

func (d *RTCMessageData) ToSessionDescription() (*webrtc.SessionDescription, error) {
	sdp := webrtc.SessionDescription{}
	offerBytes, err := json.Marshal(d.data["offer"])
	if err != nil {
		return &sdp, err
	}

	err = json.Unmarshal(offerBytes, &sdp)
	if err != nil {
		return &sdp, err
	}

	return &sdp, nil
}

type RTCHandler struct {
	rtcPeerConnection *webrtc.PeerConnection
	peerConnectionId  float64
}

func NewRTCHandler(config *webrtc.Configuration) (RTCHandler, error) {
	rtcPeerConnection, err := webrtc.NewPeerConnection(*config)
	if err != nil {
		return RTCHandler{}, err
	}

	return RTCHandler{
		rtcPeerConnection: rtcPeerConnection,
		peerConnectionId:  0,
	}, nil
}

func (handler *RTCHandler) CanOffer() bool {
	return handler.peerConnectionId == 0
}

func (handler *RTCHandler) ShouldRestart(peerConnectionId float64) bool {
	return handler.peerConnectionId != peerConnectionId
}

func (handler *RTCHandler) SetPeerConnectionId(peerConnectionId float64) {
	handler.peerConnectionId = peerConnectionId
}

func (handler *RTCHandler) StartConnection(remoteSdp *webrtc.SessionDescription, routineCoordinator *RoutineCoordinator) (*webrtc.SessionDescription, error) {

	cap := webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264, ClockRate: 90000}
	videoTrack, err := webrtc.NewTrackLocalStaticSample(cap, "video", "pion")
	if err != nil {
		log.Println(err)
		return &webrtc.SessionDescription{}, err
	}
	_, err = handler.rtcPeerConnection.AddTrack(videoTrack)
	if err != nil {
		log.Println(err)
		return &webrtc.SessionDescription{}, err
	}

	handler.rtcPeerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		log.Printf("Connection State has changed %s \n", connectionState.String())

		connectionStateDesc := connectionState.String()
		switch connectionStateDesc {
		case "connected":
			routineCoordinator.SendDroneStateChannel("land")
		case "disconnected":
		case "failed":
		case "closed":
			routineCoordinator.SendDroneStateChannel("ready")
		}

	})

	handler.rtcPeerConnection.OnDataChannel(func(dataChannel *webrtc.DataChannel) {

		dataChannel.OnOpen(func() {
			log.Println("DataChannel opened.")

			defer dataChannel.Close()
			for {
				select {
				case message := <-routineCoordinator.DataChannelMessageChannel:
					messageJson := map[string]interface{}{
						"messageType": message,
					}
					data, err := json.Marshal(messageJson)
					if err != nil {
						log.Println(err)
						continue
					}
					log.Printf("%v", messageJson)
					dataChannel.SendText(string(data))
				case <-routineCoordinator.StopSignalChannel:
					log.Println("Stop DataChannel handler")
					return
				}

			}
		})

		dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
			messageJson := make(map[string]string)
			err := json.Unmarshal(msg.Data, &messageJson)
			if err != nil {
				return
			}
			message := messageJson["command"]
			log.Println(message)
			routineCoordinator.SendDroneCommandChannel(message)
		})

	})

	if err := handler.rtcPeerConnection.SetRemoteDescription(*remoteSdp); err != nil {
		log.Println(err)
		return &webrtc.SessionDescription{}, err
	}

	answer, err := handler.rtcPeerConnection.CreateAnswer(nil)
	if err != nil {
		log.Println(err)
		return &webrtc.SessionDescription{}, err
	}

	gatherComplete := webrtc.GatheringCompletePromise(handler.rtcPeerConnection)
	if err = handler.rtcPeerConnection.SetLocalDescription(answer); err != nil {
		log.Println(err)
		return &webrtc.SessionDescription{}, err
	}

	<-gatherComplete

	go func() {

		defer handler.rtcPeerConnection.Close()
		latest := time.Now()

		for {

			select {
			case frame := <-routineCoordinator.DroneFrameChannel:
				videoTrack.WriteSample(media.Sample{
					Data: frame, Duration: time.Since(latest),
				})
				latest = time.Now()
			case <-routineCoordinator.StopSignalChannel:
				log.Println("Stop Video stream handler")
				return
			}

		}
	}()

	return &answer, nil
}
