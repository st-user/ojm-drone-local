import cv2
import threading

from aiortc import VideoStreamTrack
from av import VideoFrame


class VideoCaptureAsync:

    """
        An OpenCV VideoCapture Wrapper for reading video frames asynchronously.
    """

    def __init__(self, src):
        self.cap = cv2.VideoCapture(src)
        self.grabbed = False
        self.frame = VideoFrame(width=640, height=480).to_ndarray(
                                                        format='bgr24'
                                                      )
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

    """
        A VideoStreamTrack capturing a video stream by using VideoCaptureAsync.
    """

    kind = 'video'

    def __init__(self, src):
        super().__init__()
        self.default_frame = VideoFrame(width=640, height=480)
        self.frame_before = None
        self.acap = VideoCaptureAsync(src)
        self.acap.start()

    async def recv(self):

        pts, time_base = await self.next_timestamp()

        ret, frame = self.acap.read()

        if ret:
            new_frame = VideoFrame.from_ndarray(frame, format='bgr24')
            self.frame_before = new_frame
        else:
            if self.frame_before is not None:
                new_frame = self.frame_before
            else:
                new_frame = self.default_frame

        new_frame.pts = pts
        new_frame.time_base = time_base

        return new_frame
