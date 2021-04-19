from aiortc import RTCPeerConnection, RTCSessionDescription, RTCConfiguration, RTCIceServer


class RTCConnectionHandler:

    def __init__(self):
        self.pc = None
        self.track = None
        self.iceServerInfo = None
        self.peerConnectionId = None

    def set_ice_server_info(self, iceServerInfo):
        self.iceServerInfo = iceServerInfo

    def set_pc(self):
        if self.iceServerInfo is None:
            self.pc = RTCPeerConnection()
        else:
            config = [
                RTCIceServer(self.iceServerInfo['stun']),
                RTCIceServer(
                    self.iceServerInfo['turn'],
                    self.iceServerInfo['credentials']['username'],
                    self.iceServerInfo['credentials']['password']
                )
            ]
            self.pc = RTCPeerConnection(RTCConfiguration(config))
        return self.pc
    
    def has_pc(self):
        return self.pc is not None

    def is_connected(self):
        if self.pc is None:
            return False
        return self.pc.connectionState == 'connected'

    def should_close(self):
        if self.pc is None:
            return False
        return self.pc.connectionState in [ 'disconnected', 'failed', 'closed' ]

    def should_restart(self, peerConnectionId):
        if self.peerConnectionId is None:
            return False
        if self.peerConnectionId != peerConnectionId:
            return True
        return self.should_close()

    def set_peer_connection_id(self, peerConnectionId):
        self.peerConnectionId = peerConnectionId

    def add_track(self, track):
        self.track = track
        self.pc.addTrack(track)

    async def set_up_session_description(self, sdp, _type):
        offer = RTCSessionDescription(sdp=sdp, type=_type)
        await self.pc.setRemoteDescription(offer)
        answer = await self.pc.createAnswer()
        await self.pc.setLocalDescription(answer)
        return self.pc.localDescription