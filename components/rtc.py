from aiortc import RTCPeerConnection, RTCSessionDescription, RTCConfiguration, RTCIceServer


class RTCConnectionHandler:

    """
        A Component handling WebRTC Connection(RTCPeerConnection).
        This class provides the functionalities to manage the WebRTC Connection to the remote peer
        such as handling offer/answer, checking the connection state and so on.
    """

    def __init__(self):
        self.pc = None
        self.track = None
        self.ice_server_info = None
        self.peer_connection_id = None

    def set_ice_server_info(self, ice_server_info):
        self.ice_server_info = ice_server_info

    def set_pc(self):
        if self.ice_server_info is None:
            self.pc = RTCPeerConnection()
        else:
            config = [
                RTCIceServer(self.ice_server_info['stun']),
                RTCIceServer(
                    self.ice_server_info['turn'],
                    self.ice_server_info['credentials']['username'],
                    self.ice_server_info['credentials']['password']
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

    def should_restart(self, peer_connection_id):
        if self.peer_connection_id is None:
            return False
        if self.peer_connection_id != peer_connection_id:
            return True
        return self.should_close()

    def set_peer_connection_id(self, peer_connection_id):
        self.peer_connection_id = peer_connection_id

    def add_track(self, track):
        self.track = track
        self.pc.addTrack(track)

    async def set_up_session_description(self, sdp, _type):
        offer = RTCSessionDescription(sdp=sdp, type=_type)
        await self.pc.setRemoteDescription(offer)
        answer = await self.pc.createAnswer()
        await self.pc.setLocalDescription(answer)
        return self.pc.localDescription