import cv2
import threading

from aiortc import VideoStreamTrack
from av import VideoFrame
from typing import ClassVar, Optional, Tuple


class VideoCaptureAsync:

    """
        An OpenCV VideoCapture Wrapper for reading video frames asynchronously.
    """

    def __init__(self, src: str) -> None:
        self._cap: cv2.VideoCapture = cv2.VideoCapture(src)
        self._grabbed: bool = False
        self._frame: VideoFrame = VideoFrame(width=640, height=480).to_ndarray(
                                                        format='bgr24'
                                                      )
        self._started: bool = False
        self._read_lock: threading.Lock = threading.Lock()

    def start(self) -> None:
        self._started = True
        self._thread: threading.Thread = threading.Thread(
                                            target=self.update, args={}
                                         )
        self._thread.daemon = True
        self._thread.start()

    def update(self) -> None:
        while self._started:
            grabbed, frame = self._cap.read()
            with self._read_lock:
                self._grabbed = grabbed
                self._frame = frame

    def read(self) -> Tuple[bool, VideoFrame]:
        with self._read_lock:
            grabbed = self._grabbed
            frame = self._frame
        return grabbed, frame

    def stop(self) -> None:
        self._started = False
        self._thread.join()


class VideoCaptureTrack(VideoStreamTrack):

    """
        A VideoStreamTrack capturing a video stream by using VideoCaptureAsync.
    """

    kind: ClassVar[str] = 'video'

    def __init__(self, src: str) -> None:
        super().__init__()
        self._default_frame: VideoFrame = VideoFrame(width=640, height=480)
        self._frame_before: Optional[VideoFrame] = None
        self._acap: VideoCaptureAsync = VideoCaptureAsync(src)
        self._acap.start()

    async def recv(self) -> VideoFrame:

        pts, time_base = await self.next_timestamp()

        ret, frame = self._acap.read()

        if ret:
            new_frame = VideoFrame.from_ndarray(frame, format='bgr24')
            self._frame_before = new_frame
        else:
            if self._frame_before is not None:
                new_frame = self._frame_before
            else:
                new_frame = self._default_frame

        new_frame.pts = pts
        new_frame.time_base = time_base

        return new_frame
