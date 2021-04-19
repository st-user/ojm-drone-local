import aiohttp
import asyncio
import cv2
import json
import logging
import multiprocessing as mp
import os
import ssl
import threading
import time
import uuid

from aiohttp import web
from aiortc import VideoStreamTrack
from av import VideoFrame
from dotenv import dotenv_values

from components import drone, messaging, rtc


ROOT = os.path.dirname(__file__)


my_env = dotenv_values('.env')
logging.basicConfig(level=logging.INFO)


PORT = int(my_env['PORT'])
ROOT = os.path.dirname(__file__)
STATIC = os.path.join(ROOT, 'static')

SECRET = my_env['SECRET']
SIGNALING_ENDPOINT = my_env['SIGNALING_ENDPOINT']

USE_DRONE = drone.use()
VIDEO_SOURCE = 'udp://127.0.0.1:11111' if USE_DRONE else f'{ROOT}.resources/capture.webm'



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
  

clientMessageSocket = messaging.ClientMessageSocket()
messageChannel = messaging.MessageChannel()
rtcConnectionHandler = rtc.RTCConnectionHandler()
droneManager = drone.DroneManager()


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
    print(headers)
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
                                    droneManager.send_command_throttled_from_message(commandJson['command'])

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
                        localDescription = await rtcConnectionHandler.set_up_session_description(
                            sdp=params['sdp'], _type=params['type']
                        )

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
            time.sleep(3)


if __name__ == '__main__':
    do_main()