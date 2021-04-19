import json

class ClientMessageSocket:

    def __init__(self):
        self.ws = None

    async def send(self, data):
        if self.ws is None:
            return
        await self.ws.send_str(data)

    def set_ws(self, ws):
        self.ws = ws


class MessageChannel:

    def __init__(self):
        self.channel = None

    def send(self, messageType):
        if self.channel is None:
            return
        if self.channel.readyState == 'open':
            self.channel.send(json.dumps({
                'messageType': messageType
            }))
    
    def set_channel(self, channel):
        self.channel = channel