import json
import aiortc

from aiohttp import web
from typing import Optional


class ClientMessageSocket:

    """
        A Wrapper for websocket connection.
    """

    def __init__(self) -> None:
        self._ws: Optional[web.WebSocketResponse] = None

    async def send(self, data: str) -> None:
        if self._ws is None:
            return
        await self._ws.send_str(data)

    def set_ws(self, ws: web.WebSocketResponse) -> None:
        self._ws = ws


class MessageChannel:

    """
        A Wrapper for RTCDataChannel(WebRTC).
    """

    def __init__(self) -> None:
        self._channel: Optional[aiortc.RTCDataChannel] = None

    def send(self, message_type: str) -> None:
        if self._channel is None:
            return
        if self._channel.readyState == 'open':
            self._channel.send(json.dumps({
                'messageType': message_type
            }))

    def set_channel(self, channel: aiortc.RTCDataChannel) -> None:
        self._channel = channel
