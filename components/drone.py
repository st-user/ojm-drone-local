import asyncio
import logging
import socket
import time

from . import my_env
from typing import Any, Dict, Optional


use_drone_env = my_env.get_env('USE_DRONE', 'False')
USE_DRONE: bool = False if use_drone_env.lower() == 'false' else bool(use_drone_env)

LISTENING_IP: str = '0.0.0.0'
LISTENING_PORT: int = 8890

TELLO_IP: str = '192.168.10.1'
TELLO_PORT: str = 8889

MOVE_SPEED: int = 30
ROTATION_SPEED: int = 30

DRONE_COMMANDS: Dict[str, str] = dict()
DRONE_COMMANDS['forward'] = f'forward {MOVE_SPEED}'
DRONE_COMMANDS['back'] = f'back {MOVE_SPEED}'
DRONE_COMMANDS['right'] = f'right {MOVE_SPEED}'
DRONE_COMMANDS['left'] = f'left {MOVE_SPEED}'
DRONE_COMMANDS['up'] = f'up {MOVE_SPEED}'
DRONE_COMMANDS['down'] = f'down {MOVE_SPEED}'
DRONE_COMMANDS['cw'] = f'cw {ROTATION_SPEED}'
DRONE_COMMANDS['ccw'] = f'ccw {ROTATION_SPEED}'

logging.info(f'USE_DRONE: {USE_DRONE}')


def use() -> bool:
    return USE_DRONE


class AsyncDgramServerProtocol:

    """
        An UDP Protocol implementation for holding a received message.
    """

    _transport: asyncio.DatagramTransport
    message: str

    def connection_made(self, transport: asyncio.DatagramTransport) -> None:
        self._transport = transport

    def datagram_received(self, data: bytes, addr: str) -> None:
        self.message = data.decode()

    def connection_lost(self, exec: OSError) -> None:
        if self._transport is None:
            return
        self._transport.close()


class DroneManager:

    """
        A Component for controling a drone(Tello).
        This class provides the functionalities to manage the drone
        such as sending messages to the drone,
        receiveing the state information from the drone and so on.
    """

    def __init__(self) -> None:
        self._socket: Any = None
        self._recv_server: Optional[AsyncDgramServerProtocol] = None
        self._is_initialized: bool = False
        self._throttle_timestamp: Optional[float] = None

    def start(self) -> None:
        if self._is_initialized:
            return
        logging.info('Drone initialized.')
        command_socket = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        self._socket = command_socket
        self._is_initialized = True
        self.send_command('command')
        asyncio.ensure_future(self._ping_to_drone())
        asyncio.ensure_future(self._start_server())

    def stream_on(self) -> None:
        self.send_command('streamon')
        logging.info("streamon")

    def send_command(self, command: str) -> None:
        logging.info(f'Send command [{command}]')

        if USE_DRONE:
            self._socket.sendto(
                    command.encode('utf-8'), (TELLO_IP, TELLO_PORT)
                )

    def send_command_throttled(self, command: str) -> None:
        current_timestamp = time.perf_counter()
        if self._throttle_timestamp is None or (0.5 <= current_timestamp - self._throttle_timestamp):
            self.send_command(command)
        else:
            logging.info('Too offen. Ingnore the command[{command}].')
        self._throttle_timestamp = current_timestamp

    def send_command_throttled_from_message(self, message: str) -> None:
        self.send_command_throttled(DRONE_COMMANDS[message])

    async def _ping_to_drone(self) -> None:
        while self._is_initialized:
            await asyncio.sleep(5)
            self.send_command('command')
            self._print_drone_status()

    async def _start_server(self):
        if USE_DRONE:
            self._recv_server = AsyncDgramServerProtocol()
            loop = asyncio.get_running_loop()
            await loop.create_datagram_endpoint(
                lambda: self._recv_server,
                local_addr=(LISTENING_IP, LISTENING_PORT)
            )

    def _print_drone_status(self) -> None:
        if self._recv_server is None:
            return
        message = self._recv_server.message
        if message is None:
            return
        props = message.split(';')
        prop_dict: Dict[str, str] = dict()
        for prop in props:
            _prop = prop.split(':')
            if len(_prop) != 2:
                continue
            key = _prop[0]
            value = _prop[1]
            prop_dict[key] = value
        if 'bat' in prop_dict:
            logging.info(f"Battery level {prop_dict['bat']}%")
