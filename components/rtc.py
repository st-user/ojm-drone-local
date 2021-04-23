from . import video

from aiortc import (
    RTCPeerConnection,
    RTCSessionDescription,
    RTCConfiguration,
    RTCIceServer
)
from typing import Any, Dict, NoReturn, Optional, Union, Tuple


class RTCConnectionHandler:

    """
        A Component handling WebRTC Connection(RTCPeerConnection).
        This class provides the functionalities
        to manage the WebRTC Connection to the remote peer
        such as handling offer/answer, checking the connection state and so on.
    """

    def __init__(self) -> None:
        self._pc: Optional[RTCPeerConnection] = None
        self._track: Optional[video.VideoCaptureTrack] = None
        self._ice_server_info:  Optional[Dict[str, Any]] = None
        self._peer_connection_id: Optional[int] = None

    def set_ice_server_info(self, ice_server_info: Dict[str, Any]) -> None:
        self._ice_server_info = ice_server_info

    def set_pc(self) -> RTCPeerConnection:
        if self._ice_server_info is None:
            self._pc = RTCPeerConnection()
        else:
            config = [
                RTCIceServer(self._ice_server_info['stun']),
                RTCIceServer(
                    self._ice_server_info['turn'],
                    self._ice_server_info['credentials']['username'],
                    self._ice_server_info['credentials']['password']
                )
            ]
            self._pc = RTCPeerConnection(RTCConfiguration(config))
        return self._pc

    def has_pc(self) -> bool:
        return self._pc is not None

    def is_connected(self) -> bool:
        if self._pc is None:
            return False
        return self._pc.connectionState == 'connected'

    def should_close(self) -> bool:
        if self._pc is None:
            return False
        return self._pc.connectionState in [
                        'disconnected', 'failed', 'closed'
                    ]

    def should_restart(self, peer_connection_id: int) -> bool:
        if self._peer_connection_id is None:
            return False
        if self._peer_connection_id != peer_connection_id:
            return True
        return self.should_close()

    def set_peer_connection_id(self, peer_connection_id: int) -> None:
        self._peer_connection_id = peer_connection_id

    def add_track(self, track: video.VideoCaptureTrack) -> None:
        self._track = track
        if self._pc is not None:
            self._pc.addTrack(track)

    async def set_up_session_description(
        self, sdp: str, _type: str
    ) -> Dict[str, str]:

        if self._pc is None:
            raise ValueError('RTCPeerConnection is None')

        offer = RTCSessionDescription(sdp=sdp, type=_type)
        await self._pc.setRemoteDescription(offer)
        answer = await self._pc.createAnswer()
        await self._pc.setLocalDescription(answer)
        return {
            'sdp': self._pc.localDescription.sdp,
            'type': self._pc.localDescription.type
        }
