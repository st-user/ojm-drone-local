from aiohttp import web
import asyncio
import aiohttp
import os
import json
import uuid
import cv2
from av import VideoFrame
import socket
import threading
from time import sleep
from dotenv import dotenv_values
import ssl
import logging

from aiortc import MediaStreamTrack, VideoStreamTrack, RTCPeerConnection, RTCSessionDescription, RTCConfiguration, RTCIceServer
from aiortc.contrib.media import MediaBlackhole, MediaPlayer, MediaRecorder, MediaRelay

import multiprocessing as mp


my_env = dotenv_values('.env')
logging.basicConfig(level=logging.INFO)

USE_DRONE = False

PORT = int(my_env['PORT'])
ROOT = os.path.dirname(__file__)
STATIC = os.path.join(ROOT, 'static')

SECRET = my_env['SECRET']
SIGNALING_ENDPOINT = my_env['SIGNALING_ENDPOINT']

LISTENING_IP = '0.0.0.0'
LISTENING_PORT = 8890

TELLO_IP = '192.168.10.1'
TELLO_PORT = 8889

VIDEO_SOURCE = 'udp://127.0.0.1:11111' if USE_DRONE else f'{ROOT}.resources/capture.webm'

MOVE_SPEED = 30
ROTATION_SPEED = 30

DRONE_COMMANDS = dict()
DRONE_COMMANDS['forward'] = f'forward {MOVE_SPEED}'
DRONE_COMMANDS['back'] = f'back {MOVE_SPEED}'
DRONE_COMMANDS['right'] = f'right {MOVE_SPEED}'
DRONE_COMMANDS['left'] = f'left {MOVE_SPEED}'
DRONE_COMMANDS['up'] = f'up {MOVE_SPEED}'
DRONE_COMMANDS['down'] = f'down {MOVE_SPEED}'
DRONE_COMMANDS['cw'] = f'cw {ROTATION_SPEED}'
DRONE_COMMANDS['ccw'] = f'ccw {ROTATION_SPEED}'


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

    async def set_up_session_description(self, offer):
        await self.pc.setRemoteDescription(offer)
        answer = await self.pc.createAnswer()
        await self.pc.setLocalDescription(answer)
        return self.pc.localDescription



class VideoCaptureAsync:

    def __init__(self, src):
        self.cap = cv2.VideoCapture(src)
        self.grabbed = False
        self.frame = VideoFrame(width=640, height=480).to_ndarray(format='bgr24')
        self.started = False
        self.read_lock = threading.Lock()

    def start(self):
        self.started = True
        self.thread = threading.Thread(target=self.update, args={})
        self.thread.daemon = True
        self.thread.start()
        return self

    def update(self):
        while self.started:
            grabbed, frame = self.cap.read()
            with self.read_lock:
                self.grabbed = grabbed
                self.frame = frame
            
    def read(self):
        with self.read_lock:
            frame = self.frame
            grabbed = self.grabbed
        return grabbed, frame

    def stop(self):
        self.started = False
        self.thread.join()
       

class VideoCaptureTrack(VideoStreamTrack):

    kind = 'video'

    def __init__(self):
        super().__init__()
        self.defaultFrame = VideoFrame(width=640, height=480)
        self.frameBefore = None
        self.acap = VideoCaptureAsync(VIDEO_SOURCE)
        self.acap.start()

    async def recv(self):

        pts, time_base = await self.next_timestamp()

        ret, frame = self.acap.read()

        if ret:
            new_frame = VideoFrame.from_ndarray(frame, format='bgr24')
            self.frameBefore = new_frame
        else:
            if self.frameBefore is not None:
                new_frame = self.frameBefore
            else:    
                new_frame = self.defaultFrame

        new_frame.pts = pts
        new_frame.time_base = time_base

        return new_frame


class AsyncDgramServerProtocol:
    def connection_made(self, transport):
        self.transport = transport

    def datagram_received(self, data, addr):
        self.message = data.decode()
    
    def connection_lost(self, exec):
        if self.transport is None:
            return
        self.transport.close()


class DroneManager:

    def __init__(self):
        self.socket = None
        self.recv_server = None
        self.isInitialized = False

    def start(self):
        if self.isInitialized:
            return
        logging.info('Drone initialized.')
        command_socket = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        self.socket = command_socket
        self.isInitialized = True
        self.send_command('command')
        asyncio.ensure_future(self.ping_to_drone())
        asyncio.ensure_future(self.start_server())

    def stream_on(self):
        self.send_command('streamon')
        logging.info("streamon")

    async def ping_to_drone(self):
        while self.isInitialized:
            await asyncio.sleep(5)
            self.send_command('command')
            self.print_drone_status()

    async def start_server(self):
        if USE_DRONE:
            self.recv_server = AsyncDgramServerProtocol()
            loop = asyncio.get_running_loop()
            await loop.create_datagram_endpoint(
                lambda: self.recv_server,
                local_addr=(LISTENING_IP, LISTENING_PORT)
            )
    
    def print_drone_status(self):
        if self.recv_server is None:
            return
        message = self.recv_server.message
        if message is None:
            return
        props = message.split(';')
        propDict = dict()
        for prop in props:
            _prop = prop.split(':')
            if len(_prop) != 2:
                continue
            key = _prop[0]
            value = _prop[1]
            propDict[key] = value
        if 'bat' in propDict:
            logging.info(f"Battery level {propDict['bat']}%")
        

    def send_command(self, command):
        logging.info(f'Send command [{command}]')  

        if USE_DRONE:
            self.socket.sendto(command.encode('utf-8'), (TELLO_IP, TELLO_PORT))

    

clientMessageSocket = ClientMessageSocket()
messageChannel = MessageChannel()
rtcConnectionHandler = RTCConnectionHandler()
droneManager = DroneManager()


def get_static_content(filename, mediaType):
    content = open(os.path.join(STATIC, filename), 'r').read()
    return web.Response(content_type=mediaType, text=content)

def log_request(endpointName):
    logging.info(f'Request to {endpointName}')

async def index(request):
    if rtcConnectionHandler.has_pc():
        dataQueue.put(os.getpid())
    return get_static_content('index.html', 'text/html')


async def javascript(request):
    return get_static_content('main.js', 'application/javascript')


async def stylesheet(request):
    return get_static_content('main.css', 'text/css')


async def request_key():
    headers = { 'Authorization': f'bearer {SECRET}' }
    async with aiohttp.ClientSession() as session:
        ssl_context = ssl.create_default_context(ssl.Purpose.CLIENT_AUTH)
        async with session.get(f'{SIGNALING_ENDPOINT}/generateKey', headers=headers, ssl=ssl_context) as response:

            logging.info(f'Status: {response.status:d}')
            logging.info(f"Content-type: {response.headers['content-type']}")

            return await response.json()


async def start_signaling(startKey):
    async with aiohttp.ClientSession() as session:
        ssl_context = ssl.create_default_context(ssl.Purpose.CLIENT_AUTH)
        async with session.ws_connect(f'{SIGNALING_ENDPOINT}/signaling?startKey={startKey}', ssl=ssl_context) as ws:
            async for msg in ws:
                if msg.type == aiohttp.WSMsgType.TEXT:

                    dataJson = json.loads(msg.data)
                    messageType = dataJson['messageType']

                    if messageType == 'iceServerInfo':
                        if 'iceServerInfo' in dataJson:
                            iceServerInfo = dataJson['iceServerInfo']
                            rtcConnectionHandler.set_ice_server_info(iceServerInfo)

                    if messageType == 'canOffer':
                        canOffer = True
                        peerConnectionId = dataJson['peerConnectionId']

                        if rtcConnectionHandler.has_pc():
                            canOffer = False
                            logging.warn('Connection has already bean created.')

                        if rtcConnectionHandler.should_restart(peerConnectionId):
                            canOffer = True
                            waiting = True
                            dataQueue.put(os.getpid())
                            logging.warn('Connection is not available. So retry.')
                        
                        logging.debug(f'----- peerConnectionId ----- {peerConnectionId}')
                        rtcConnectionHandler.set_peer_connection_id(peerConnectionId)
                        await ws.send_str(json.dumps(
                            {
                                'messageType': 'canOffer',
                                'canOffer': canOffer
                            }
                        ))
                            

                    if messageType == 'offer':

                        params = dataJson['offer']
                        offer = RTCSessionDescription(sdp=params['sdp'], type=params['type'])

                        pc_id = f'PeerConnection({uuid.uuid4()})' 
                        def log_info(msg):
                            logging.info(f'{pc_id} {msg}')

                        pc = rtcConnectionHandler.set_pc()
      
                        @pc.on('datachannel')
                        def on_datachannel(channel):
                            messageChannel.set_channel(channel)
                            @channel.on('message')
                            def on_message(message):
                                if isinstance(message, str):
                                    commandJson = json.loads(message)
                                    command = commandJson['command']
                                    droneManager.send_command(DRONE_COMMANDS[command])

                        @pc.on('connectionstatechange')
                        async def on_connectionstatechange():
                            log_info(f'Connection state is {pc.connectionState}')

                            if rtcConnectionHandler.is_connected():
                                await clientMessageSocket.send(json.dumps({
                                    'messageType': 'stateChange',
                                    'state': 'land'
                                }))


                            if rtcConnectionHandler.should_close():
                                await clientMessageSocket.send(json.dumps({
                                    'messageType': 'stateChange',
                                    'state': 'ready'
                                }))

                        droneManager.start()
                        droneManager.stream_on()

                        rtcConnectionHandler.add_track(VideoCaptureTrack())
                        localDescription = await rtcConnectionHandler.set_up_session_description(offer)

                        await ws.send_str(json.dumps(
                            {
                                'messageType': 'answer',
                                'answer': {"sdp": localDescription.sdp, "type": localDescription.type}
                            }
                        ))

                elif msg.type == aiohttp.WSMsgType.ERROR:
                    logging.error('error')
                    break


async def generate_key(request):
    log_request('generateKey')

    result = await request_key()
    startKey = result['startKey']
    content = json.dumps({ 'startKey': startKey })
    return web.Response(content_type='application/json', text=content)


async def start_app(request):
    log_request('startApp')

    params = await request.json()
    startKey = params['startKey']
    asyncio.ensure_future(start_signaling(startKey))
    return web.Response(content_type='application/json', text=r"{}")


async def health_check(request):
    log_request('healthCheck')

    if waiting:
        return web.Response(content_type='application/json', status=503, text=r"{}")
    return web.Response(content_type='application/json', text=r"{}")

async def takeoff(request):
    log_request('takeoff')

    result = droneManager.send_command('takeoff')
    messageChannel.send('takeoff')

    return web.Response(content_type='application/json', text=r"{}")


async def land(request):
    log_request('land')

    result = droneManager.send_command('land')
    messageChannel.send('land')

    return web.Response(content_type='application/json', text=r"{}")


async def client_ws_handler(request):

    ws = web.WebSocketResponse()
    await ws.prepare(request)

    clientMessageSocket.set_ws(ws)
    async for msg in ws:
        if msg.type == aiohttp.WSMsgType.TEXT:
            logging.warn(f'This client does not handle incoming messages. --{msg.data}--')
            


def main(_dataQueue):

    global dataQueue
    global waiting
    dataQueue = _dataQueue
    waiting = False
    logging.info(f'Subproc pid {os.getpid():d}')

    app = web.Application()
    app.router.add_get('/', index)
    app.router.add_get('/main.js', javascript)
    app.router.add_get('/main.css', stylesheet)

    app.router.add_get('/generateKey', generate_key)
    app.add_routes([web.get('/state', client_ws_handler)])
    app.router.add_post('/startApp', start_app)
    app.router.add_get('/healthCheck', health_check)
    app.router.add_get('/takeoff', takeoff)
    app.router.add_get('/land', land)   

    web.run_app(
        app, access_log=None, host='0.0.0.0', port=PORT
    )

def do_main():
    while True:
        dataQueue = mp.Queue()
        p = mp.Process(target=main, args=(dataQueue,))
        try:
            logging.info('Start proc.')
            p.start()
            stoppedPid = dataQueue.get()
            logging.info(f'End proc({stoppedPid:d}).')
            p.kill()
        finally:
            logging.info('Kill proc.')
            sleep(3)


if __name__ == '__main__':
    do_main()