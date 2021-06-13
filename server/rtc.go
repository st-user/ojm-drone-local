package main

import (
	"encoding/json"
	"errors"
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
	PeerConnectionId string
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
		PeerConnectionId: d.data["peerConnectionId"].(string),
		IsPrimary:        d.data["isPrimary"].(bool),
	}
}

func (d *RTCMessageData) ToPeerConnectionId() string {
	return d.data["peerConnectionId"].(string)
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
	peerConnectionId        string
	audiencePeerConnections map[string]AudiencePeerInfo
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
		peerConnectionId:        "",
		audiencePeerConnections: make(map[string]AudiencePeerInfo),
	}, nil
}

func (handler *RTCHandler) DecidePeerState(peerType PeerType) string {
	if peerType.IsPrimary {
		switch handler.peerConnectionId {
		case "":
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

func (handler *RTCHandler) IsPrimary(peerConnectionId string) bool {
	return handler.peerConnectionId == peerConnectionId
}

func (handler *RTCHandler) StartPrimaryConnection(
	remoteSdp *webrtc.SessionDescription,
	routineCoordinator *RoutineCoordinator) (*webrtc.SessionDescription, error) {

	cap := webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264, ClockRate: 90000}
	videoTrack, err := webrtc.NewTrackLocalStaticSample(cap, "video", "pion")
	if err != nil {
		Log.Info("%v", err)
		return &webrtc.SessionDescription{}, err
	}
	handler.videoTrack = videoTrack

	rtpSender, err := handler.rtcPeerConnection.AddTrack(videoTrack)
	if err != nil {
		Log.Info("%v", err)
		return &webrtc.SessionDescription{}, err
	}

	go func() {
		rtcpBuf := make([]byte, 1500)
		for {
			select {
			case <-routineCoordinator.StopSignalChannel:
				Log.Info("Stops WebRTC event loop.")
				return
			default:

				n, _, rtcpErr := rtpSender.Read(rtcpBuf)
				if rtcpErr != nil {
					continue
				}
				rtcpPacket := rtcpBuf[:n]

				pkts, err := rtcp.Unmarshal(rtcpPacket)
				if err != nil {
					Log.Info("%v", err)
					continue
				}

				for _, pkt := range pkts {
					routineCoordinator.SendRTCPPacketChannel(pkt)
				}
			}
		}
	}()

	handler.rtcPeerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		Log.Info("Connection State has changed %s \n", connectionState.String())

		connectionStateDesc := connectionState.String()
		switch connectionStateDesc {
		case "connected":
			routineCoordinator.SendDroneStateChannel("land")
		case "disconnected":
			fallthrough
		case "failed":
			fallthrough
		case "closed":
			routineCoordinator.SendDroneStateChannel("ready")
		}

	})

	handler.rtcPeerConnection.OnDataChannel(func(dataChannel *webrtc.DataChannel) {

		dataChannel.OnOpen(func() {
			Log.Info("DataChannel opened.")

			defer dataChannel.Close()
			for {
				select {
				case message := <-routineCoordinator.DataChannelMessageChannel:
					messageJson := map[string]interface{}{
						"messageType": message,
					}
					data, err := json.Marshal(messageJson)
					if err != nil {
						Log.Info("%v", err)
						continue
					}
					dataChannel.SendText(string(data))
				case <-routineCoordinator.StopSignalChannel:
					Log.Info("Stop handling dataChannel.")
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
			Log.Info("%v", message)
			routineCoordinator.SendDroneCommandChannel(DroneCommand{
				CommandType: "vector",
				Command:     message,
			})
		})

	})

	if err := handler.rtcPeerConnection.SetRemoteDescription(*remoteSdp); err != nil {
		Log.Info("%v", err)
		return &webrtc.SessionDescription{}, err
	}

	answer, err := handler.rtcPeerConnection.CreateAnswer(nil)
	if err != nil {
		Log.Info("%v", err)
		return &webrtc.SessionDescription{}, err
	}

	gatherComplete := webrtc.GatheringCompletePromise(handler.rtcPeerConnection)
	if err = handler.rtcPeerConnection.SetLocalDescription(answer); err != nil {
		Log.Info("%v", err)
		return &webrtc.SessionDescription{}, err
	}

	<-gatherComplete

	go func() {

		<-routineCoordinator.StopSignalChannel

		handler.rtcPeerConnection.Close()
	}()

	go func() {

		latest := time.Now()

		for {

			select {
			case frame := <-routineCoordinator.DroneFrameChannel:
				videoTrack.WriteSample(media.Sample{
					Data: frame, Duration: time.Since(latest),
				})
				latest = time.Now()
			case <-routineCoordinator.StopSignalChannel:
				Log.Info("Stop sending video stream.")
				return
			}

		}
	}()

	return handler.rtcPeerConnection.LocalDescription(), nil
}

func (handler *RTCHandler) StartAudienceConnection(
	peerConnectionId string,
	remoteSdp *webrtc.SessionDescription,
	routineCoordinator *RoutineCoordinator) (*webrtc.SessionDescription, error) {

	if handler.videoTrack == nil {
		return nil, errors.New("videoTrack is nil")
	}

	peerConnection, err := webrtc.NewPeerConnection(*handler.config)
	if err != nil {
		Log.Info("%v", err)
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
		Log.Info("%v", err)
		return &webrtc.SessionDescription{}, err
	}

	terminate := func() {
		peerConnection.Close()
		delete(handler.audiencePeerConnections, peerConnectionId)
	}

	go func() {

		select {
		case <-peerInfo.audienceRTCStopChannel:
			terminate()
		case <-routineCoordinator.StopSignalChannel:
			terminate()
		}

	}()

	go func() {
		rtcpBuf := make([]byte, 1500)

		for {

			select {
			case <-peerInfo.audienceRTCStopChannel:
				Log.Info("Stops an audience WebRTC event loop. %v", peerConnectionId)
				return
			case <-routineCoordinator.StopSignalChannel:
				Log.Info("Stop audiences WebRTC event loop.")
				return
			default:

				n, _, rtcpErr := rtpSender.Read(rtcpBuf)
				if rtcpErr != nil {
					continue
				}
				rtcpPacket := rtcpBuf[:n]

				pkts, err := rtcp.Unmarshal(rtcpPacket)
				if err != nil {
					Log.Info("%v", err)
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
		Log.Info("%v", err)
		return &webrtc.SessionDescription{}, err
	}

	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		Log.Info("%v", err)
		return &webrtc.SessionDescription{}, err
	}

	gatherComplete := webrtc.GatheringCompletePromise(peerConnection)

	err = peerConnection.SetLocalDescription(answer)
	if err != nil {
		Log.Info("%v", err)
		return &webrtc.SessionDescription{}, err
	}

	<-gatherComplete

	return peerConnection.LocalDescription(), nil
}

func (handler *RTCHandler) SendAudienceRTCStopChannel(peerConnectionId string) {
	con, ok := handler.audiencePeerConnections[peerConnectionId]
	if ok && con.audienceRTCStopChannel != nil {
		close(con.audienceRTCStopChannel)
	}
}

func (handler *RTCHandler) DeleteAudience(peerConnectionId string) {
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
