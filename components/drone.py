import asyncio
import logging
import socket
import time


USE_DRONE = False


LISTENING_IP = '0.0.0.0'
LISTENING_PORT = 8890

TELLO_IP = '192.168.10.1'
TELLO_PORT = 8889

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


def use():
    return USE_DRONE


class AsyncDgramServerProtocol:

    """
        An UDP Protocol implementation for holding a received message.
    """

    def connection_made(self, transport):
        self.transport = transport

    def datagram_received(self, data, addr):
        self.message = data.decode()

    def connection_lost(self, exec):
        if self.transport is None:
            return
        self.transport.close()


class DroneManager:

    """
        A Component for controling a drone(Tello).
        This class provides the functionalities to manage the drone
        such as sending messages to the drone,
        receiveing the state information from the drone and so on.
    """

    def __init__(self):
        self.socket = None
        self.recv_server = None
        self.is_initialized = False
        self.throttle_timestamp = None

    def start(self):
        if self.is_initialized:
            return
        logging.info('Drone initialized.')
        command_socket = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        self.socket = command_socket
        self.is_initialized = True
        self.send_command('command')
        asyncio.ensure_future(self.ping_to_drone())
        asyncio.ensure_future(self.start_server())

    def stream_on(self):
        self.send_command('streamon')
        logging.info("streamon")

    async def ping_to_drone(self):
        while self.is_initialized:
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
        prop_dict = dict()
        for prop in props:
            _prop = prop.split(':')
            if len(_prop) != 2:
                continue
            key = _prop[0]
            value = _prop[1]
            prop_dict[key] = value
        if 'bat' in prop_dict:
            logging.info(f"Battery level {prop_dict['bat']}%")

    def send_command(self, command):
        logging.info(f'Send command [{command}]')

        if USE_DRONE:
            self.socket.sendto(command.encode('utf-8'), (TELLO_IP, TELLO_PORT))

    def send_command_throttled(self, command):
        current_timestamp = time.perf_counter()
        if self.throttle_timestamp is None or (0.5 <= current_timestamp - self.throttle_timestamp):
            self.send_command(command)
        else:
            logging.info('Too offen. Ingnore the command[{command}].')
        self.throttle_timestamp = current_timestamp

    def send_command_throttled_from_message(self, message):
        self.send_command_throttled(DRONE_COMMANDS[message])
