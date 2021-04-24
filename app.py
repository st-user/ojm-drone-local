import aiohttp
import asyncio
import json
import logging
import os
import ssl
import uuid
import multiprocessing as mp

from aiohttp import web


from components import drone, messaging, rtc, video, my_env
from typing import Any, Dict


PORT: int = int(my_env.get_env('PORT', '8000'))
ROOT: str = os.path.dirname(__file__)
STATIC: str = os.path.join(ROOT, 'static')

SECRET: str = my_env.get_env('SECRET')
SIGNALING_ENDPOINT: str = my_env.get_env('SIGNALING_ENDPOINT', 'http://localhost:8080')

USE_DRONE: bool = drone.use()
VIDEO_SOURCE: str = os.path.join(ROOT, '.resources', 'capture.webm')

if USE_DRONE:
    VIDEO_SOURCE = 'udp://127.0.0.1:11111'


logging.info(r"Loads 'app.py' module")


# Initialize the components.
client_message_socket: messaging.ClientMessageSocket = messaging.ClientMessageSocket()
message_channel: messaging.MessageChannel = messaging.MessageChannel()
rtc_connection_handler: rtc.RTCConnectionHandler = rtc.RTCConnectionHandler()
drone_manager: drone.DroneManager = drone.DroneManager()

data_queue: mp.Queue
waiting: bool


# Methods for handling HTTP Requests and WebSocket connections.
def log_request(endpoint_name: str) -> None:
    logging.info(f'Request to {endpoint_name}')


def get_static_content(filename: str, media_type: str) -> web.Response:
    with open(os.path.join(STATIC, filename), 'r') as f:
        content = f.read()
        return web.Response(content_type=media_type, text=content)


async def index(request: web.Request) -> web.Response:
    if rtc_connection_handler.has_pc():
        waiting = True

        def _stop() -> None:
            data_queue.put(os.getpid())
        asyncio.get_running_loop().call_later(1, _stop)

        return get_static_content('waiting.html', 'text/html')
    return get_static_content('index.html', 'text/html')


async def javascript(request: web.Request) -> web.Response:
    return get_static_content('main.js', 'application/javascript')


async def stylesheet(request: web.Request) -> web.Response:
    return get_static_content('main.css', 'text/css')


async def request_key() -> Dict[str, Any]:
    headers = {'Authorization': f'bearer {SECRET}'}
    async with aiohttp.ClientSession() as session:
        ssl_context = ssl.create_default_context(ssl.Purpose.CLIENT_AUTH)
        endpoint_url = f'{SIGNALING_ENDPOINT}/generateKey'
        async with session.get(
                                endpoint_url,
                                headers=headers,
                                ssl=ssl_context) as response:

            logging.info(f'Status: {response.status:d}')
            logging.info(f"Content-type: {response.headers['content-type']}")

            return await response.json()


async def start_signaling_connection(
                                        session: aiohttp.ClientSession,
                                        startKey: str,
                                        connection_result_future: asyncio.Future
                                    ) -> None:

    ssl_context = ssl.create_default_context(ssl.Purpose.CLIENT_AUTH)
    endpoint_url = f'{SIGNALING_ENDPOINT}/signaling?startKey={startKey}'
    async with session.ws_connect(endpoint_url, ssl=ssl_context) as ws:

        connection_result_future.set_result(True)

        async for msg in ws:
            if msg.type == aiohttp.WSMsgType.TEXT:

                data_json = json.loads(msg.data)
                message_type = data_json['messageType']

                if message_type == 'iceServerInfo':
                    if 'iceServerInfo' in data_json:
                        ice_server_info = data_json['iceServerInfo']
                        rtc_connection_handler.set_ice_server_info(
                                                    ice_server_info
                                              )

                if message_type == 'canOffer':
                    can_offer = True
                    peer_connection_id = data_json['peerConnectionId']

                    if rtc_connection_handler.has_pc():
                        can_offer = False
                        logging.warn('Connection has already bean created.')

                    if rtc_connection_handler.should_restart(
                                                peer_connection_id
                                              ):
                        can_offer = True
                        waiting = True
                        data_queue.put(os.getpid())
                        logging.warn('Connection is not available. So retry.')

                    logging.debug(
                        f'----- peer_connection_id ----- {peer_connection_id}'
                    )
                    rtc_connection_handler.set_peer_connection_id(
                                                peer_connection_id
                                          )
                    await ws.send_str(json.dumps(
                        {
                            'messageType': 'canOffer',
                            'canOffer': can_offer
                        }
                    ))

                if message_type == 'offer':

                    params = data_json['offer']

                    """ Debug code
                    sdp = params['sdp']
                    sdps = sdp.split(r'\r\n')
                    for _sdp in sdps:
                        print(_sdp)
                    """

                    pc_id = f'PeerConnection({uuid.uuid4()})'

                    def log_info(msg):
                        logging.info(f'{pc_id} {msg}')

                    pc = rtc_connection_handler.set_pc()

                    @pc.on('datachannel')
                    def on_datachannel(channel):
                        message_channel.set_channel(channel)

                        @channel.on('message')
                        def on_message(message):
                            if isinstance(message, str):
                                command_json = json.loads(message)
                                drone_manager.send_command_throttled_from_message(
                                                command_json['command']
                                             )

                    @pc.on('connectionstatechange')
                    async def on_connectionstatechange():
                        log_info(f'Connection state is {pc.connectionState}')

                        if rtc_connection_handler.is_connected():
                            await client_message_socket.send(json.dumps({
                                'messageType': 'stateChange',
                                'state': 'land'
                            }))

                        if rtc_connection_handler.should_close():
                            await client_message_socket.send(json.dumps({
                                'messageType': 'stateChange',
                                'state': 'ready'
                            }))

                    drone_manager.start()
                    drone_manager.stream_on()

                    rtc_connection_handler.add_track(
                                        video.VideoCaptureTrack(VIDEO_SOURCE)
                                    )
                    local_description = await rtc_connection_handler.set_up_session_description(
                                                    sdp=params['sdp'], _type=params['type']
                                                )

                    await ws.send_str(json.dumps(
                        {
                            'messageType': 'answer',
                            'answer': {
                                        'sdp': local_description['sdp'],
                                        'type': local_description['type']
                                      }
                        }))

            elif msg.type == aiohttp.WSMsgType.ERROR:
                logging.error('error')
                break


async def start_signaling(start_key: str, connection_result_future: asyncio.Future) -> None:
    async with aiohttp.ClientSession() as session:
        try:
            await start_signaling_connection(
                        session, start_key, connection_result_future
            )
        except aiohttp.client_exceptions.WSServerHandshakeError as e:
            logging.error(e)
            connection_result_future.set_result(False)


async def generate_key(request: web.Request) -> web.Response:
    log_request('generateKey')

    result = await request_key()
    start_key = result['startKey']
    content = json.dumps({'startKey': start_key})
    return web.Response(content_type='application/json', text=content)


async def start_app(request: web.Request) -> web.Response:
    log_request('startApp')

    params = await request.json()
    start_key = params['startKey']
    connection_result_future = asyncio.get_running_loop().create_future()

    asyncio.ensure_future(start_signaling(start_key, connection_result_future))
    is_success = await connection_result_future
    if is_success:
        return web.Response(content_type='application/json', text=r"{}")
    else:
        return web.Response(
                        content_type='application/json', status=500, text=r"{}"
                  )


async def health_check(request: web.Request) -> web.Response:
    log_request('healthCheck')

    if waiting:
        return web.Response(
                        content_type='application/json', status=503, text=r"{}"
                  )
    return web.Response(content_type='application/json', text=r"{}")


async def takeoff(request: web.Request) -> web.Response:
    log_request('takeoff')

    drone_manager.send_command('takeoff')
    message_channel.send('takeoff')

    return web.Response(content_type='application/json', text=r"{}")


async def land(request: web.Request) -> web.Response:
    log_request('land')

    drone_manager.send_command('land')
    message_channel.send('land')

    return web.Response(content_type='application/json', text=r"{}")


async def client_ws_handler(request: web.Request):

    ws = web.WebSocketResponse()
    await ws.prepare(request)

    client_message_socket.set_ws(ws)
    async for msg in ws:
        if msg.type == aiohttp.WSMsgType.TEXT:
            logging.warn(
                f'This client does not handle incoming messages.'
                + ' --{msg.data}--'
            )


# Main routines.

def main(_data_queue: mp.Queue) -> None:

    """
        The main routine of this application.
    """

    global data_queue
    global waiting
    data_queue = _data_queue
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

    waiting = False
    logging.info(f'End setting up routings.')

    web.run_app(
        app, access_log=None, host='0.0.0.0', port=PORT
    )
