package main

import (
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
)

const (
	PEER_STATE_SAME  = "SAME"
	PEER_STATE_EXIST = "EXIST"
	PEER_STATE_EMPTY = "EMPTY"
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

type PeerType struct {
	PeerConnectionId float64
	IsPrimary        bool
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

func (d *RTCMessageData) ToPeerType() PeerType {
	return PeerType{
		PeerConnectionId: d.data["peerConnectionId"].(float64),
		IsPrimary:        d.data["isPrimary"].(bool),
	}
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
	rtcPeerConnection       *webrtc.PeerConnection
	config                  *webrtc.Configuration
	peerConnectionId        float64
	audiencePeerConnections map[float64]AudiencePeerInfo
	videoTrack              *webrtc.TrackLocalStaticSample
}

type AudiencePeerInfo struct {
	rtcPeerConnection      *webrtc.PeerConnection
	audienceRTCStopChannel chan struct{}
}

func NewRTCHandler(config *webrtc.Configuration) (RTCHandler, error) {
	rtcPeerConnection, err := webrtc.NewPeerConnection(*config)
	if err != nil {
		return RTCHandler{}, err
	}

	return RTCHandler{
		rtcPeerConnection:       rtcPeerConnection,
		config:                  config,
		peerConnectionId:        0,
		audiencePeerConnections: make(map[float64]AudiencePeerInfo),
	}, nil
}

func (handler *RTCHandler) DecidePeerState(peerType PeerType) string {
	if peerType.IsPrimary {
		switch handler.peerConnectionId {
		case 0:
			handler.peerConnectionId = peerType.PeerConnectionId
			return PEER_STATE_EMPTY
		case peerType.PeerConnectionId:
			return PEER_STATE_SAME
		default:
			return PEER_STATE_EXIST
		}

	} else {
		_, contains := handler.audiencePeerConnections[peerType.PeerConnectionId]
		if contains {
			return PEER_STATE_SAME
		} else {
			handler.audiencePeerConnections[peerType.PeerConnectionId] = AudiencePeerInfo{}
			return PEER_STATE_EMPTY
		}
	}
}

func (handler *RTCHandler) IsPrimary(peerConnectionId float64) bool {
	return handler.peerConnectionId == peerConnectionId
}

func (handler *RTCHandler) StartPrimaryConnection(
	remoteSdp *webrtc.SessionDescription,
	routineCoordinator *RoutineCoordinator) (*webrtc.SessionDescription, error) {

	cap := webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264, ClockRate: 90000}
	videoTrack, err := webrtc.NewTrackLocalStaticSample(cap, "video", "pion")
	if err != nil {
		log.Println(err)
		return &webrtc.SessionDescription{}, err
	}
	handler.videoTrack = videoTrack

	rtpSender, err := handler.rtcPeerConnection.AddTrack(videoTrack)
	if err != nil {
		log.Println(err)
		return &webrtc.SessionDescription{}, err
	}

	go func() {
		rtcpBuf := make([]byte, 1500)
		for {
			select {
			case <-routineCoordinator.StopSignalChannel:
				log.Println("Stops WebRTC event loop.")
				return
			default:

				n, _, rtcpErr := rtpSender.Read(rtcpBuf)
				if rtcpErr != nil {
					continue
				}
				rtcpPacket := rtcpBuf[:n]

				pkts, err := rtcp.Unmarshal(rtcpPacket)
				if err != nil {
					log.Println(err)
					continue
				}

				for _, pkt := range pkts {
					routineCoordinator.SendRTCPPacketChannel(pkt)
				}
			}
		}
	}()

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
					log.Println("Stop handling dataChannel.")
					return
				}

			}
		})

		dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
			messageJson := make(map[string]MotionVector)
			err := json.Unmarshal(msg.Data, &messageJson)
			if err != nil {
				return
			}
			message := messageJson["command"]
			log.Println(message)
			routineCoordinator.SendDroneCommandChannel(DroneCommand{
				CommandType: "vector",
				Command:     message,
			})
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
				log.Println("Stop sending video stream.")
				return
			}

		}
	}()

	return handler.rtcPeerConnection.LocalDescription(), nil
}

func (handler *RTCHandler) StartAudienceConnection(
	peerConnectionId float64,
	remoteSdp *webrtc.SessionDescription,
	routineCoordinator *RoutineCoordinator) (*webrtc.SessionDescription, error) {

	if handler.videoTrack == nil {
		return nil, errors.New("videoTrack is nil")
	}

	peerConnection, err := webrtc.NewPeerConnection(*handler.config)
	if err != nil {
		log.Println(err)
		return &webrtc.SessionDescription{}, err
	}

	stopChan := make(chan struct{})
	peerInfo := AudiencePeerInfo{
		rtcPeerConnection:      peerConnection,
		audienceRTCStopChannel: stopChan,
	}
	handler.audiencePeerConnections[peerConnectionId] = peerInfo

	rtpSender, err := peerConnection.AddTrack(handler.videoTrack)
	if err != nil {
		log.Println(err)
		return &webrtc.SessionDescription{}, err
	}

	go func() {

		<-peerInfo.audienceRTCStopChannel
		rtpSender.Stop()
	}()

	go func() {
		rtcpBuf := make([]byte, 1500)

		defer func() {
			peerConnection.Close()
			delete(handler.audiencePeerConnections, peerConnectionId)
		}()

		for {

			select {
			case <-peerInfo.audienceRTCStopChannel:
				log.Printf("Stops an audience WebRTC event loop. %v", peerConnectionId)
				return
			case <-routineCoordinator.StopSignalChannel:
				log.Println("Stop audiences WebRTC event loop.")
				return
			default:

				n, _, rtcpErr := rtpSender.Read(rtcpBuf)
				if rtcpErr != nil {
					continue
				}
				rtcpPacket := rtcpBuf[:n]

				pkts, err := rtcp.Unmarshal(rtcpPacket)
				if err != nil {
					log.Println(err)
					continue
				}

				for _, pkt := range pkts {
					_, ok := pkt.(*rtcp.PictureLossIndication)
					if ok {
						routineCoordinator.SendRTCPPacketChannel(pkt)
					}
				}
			}
		}
	}()

	err = peerConnection.SetRemoteDescription(*remoteSdp)
	if err != nil {
		log.Println(err)
		return &webrtc.SessionDescription{}, err
	}

	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		log.Println(err)
		return &webrtc.SessionDescription{}, err
	}

	gatherComplete := webrtc.GatheringCompletePromise(peerConnection)

	err = peerConnection.SetLocalDescription(answer)
	if err != nil {
		log.Println(err)
		return &webrtc.SessionDescription{}, err
	}

	<-gatherComplete

	return peerConnection.LocalDescription(), nil
}

func (handler *RTCHandler) SendAudienceRTCStopChannel(peerConnectionId float64) {
	con, ok := handler.audiencePeerConnections[peerConnectionId]
	if ok && con.audienceRTCStopChannel != nil {
		close(con.audienceRTCStopChannel)
	}
}

func (handler *RTCHandler) DeleteAudience(peerConnectionId float64) {
	audienceInfo, ok := handler.audiencePeerConnections[peerConnectionId]
	if ok {
		delete(handler.audiencePeerConnections, peerConnectionId)
		if audienceInfo.audienceRTCStopChannel != nil {
			close(audienceInfo.audienceRTCStopChannel)
		}
		if audienceInfo.rtcPeerConnection != nil {
			audienceInfo.rtcPeerConnection.Close()
		}
	}

}