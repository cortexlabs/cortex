import youtube_dl
import ffmpeg
import numpy as np
import cv2
import uuid

from pathlib import Path
from typing import Iterable, Tuple


def download_from_youtube(url: str, min_height: int) -> Path:
    target = f"{uuid.uuid1()}.mp4"
    ydl_opts = {
        "outtmpl": target,
        "format": f"worstvideo[vcodec=vp9][height>={min_height}]",
    }
    with youtube_dl.YoutubeDL(ydl_opts) as ydl:
        ydl.download([url])
    # we need to glob in case youtube-dl adds suffix
    (path,) = Path().absolute().glob(f"{target}*")
    return path


def frame_reader(path: Path, size: Tuple[int, int]) -> Iterable[np.ndarray]:
    width, height = size
    # letterbox frames to fixed size
    process = (
        ffmpeg.input(path)
        .filter("scale", size=f"{width}:{height}", force_original_aspect_ratio="decrease")
        # Negative values for x and y center the padded video
        .filter("pad", height=height, width=width, x=-1, y=-1)
        .output("pipe:", format="rawvideo", pix_fmt="rgb24")
        .run_async(pipe_stdout=True)
    )

    while True:
        in_bytes = process.stdout.read(height * width * 3)
        if not in_bytes:
            process.wait()
            break
        frame = np.frombuffer(in_bytes, np.uint8).reshape([height, width, 3])
        yield frame


class FrameWriter:
    def __init__(self, path: Path, size: Tuple[int, int]):
        width, height = size
        self.process = (
            ffmpeg.input("pipe:", format="rawvideo", pix_fmt="rgb24", s=f"{width}x{height}")
            .output(path, pix_fmt="yuv420p")
            .overwrite_output()
            .run_async(pipe_stdin=True)
        )

    def write(self, frame: np.ndarray):
        self.process.stdin.write(frame.astype(np.uint8).tobytes())

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc_value, traceback):
        self.__del__()

    def __del__(self):
        self.process.stdin.close()
        self.process.wait()


def nms(dets: np.ndarray, scores: np.ndarray, thresh: float) -> np.ndarray:
    x1 = dets[:, 0]
    y1 = dets[:, 1]
    x2 = dets[:, 2]
    y2 = dets[:, 3]

    areas = (x2 - x1 + 1) * (y2 - y1 + 1)
    order = scores.argsort()[::-1]  # get boxes with more ious first

    keep = []
    while order.size > 0:
        i = order[0]  # pick maxmum iou box
        keep.append(i)
        xx1 = np.maximum(x1[i], x1[order[1:]])
        yy1 = np.maximum(y1[i], y1[order[1:]])
        xx2 = np.minimum(x2[i], x2[order[1:]])
        yy2 = np.minimum(y2[i], y2[order[1:]])

        w = np.maximum(0.0, xx2 - xx1 + 1)  # maximum width
        h = np.maximum(0.0, yy2 - yy1 + 1)  # maxiumum height
        inter = w * h
        ovr = inter / (areas[i] + areas[order[1:]] - inter)

        inds = np.where(ovr <= thresh)[0]
        order = order[inds + 1]

    return np.array(keep).astype(np.int)


def boxes_yolo_to_xyxy(boxes: np.ndarray):
    boxes[:, 0] -= boxes[:, 2] / 2
    boxes[:, 1] -= boxes[:, 3] / 2
    boxes[:, 2] = boxes[:, 2] + boxes[:, 0]
    boxes[:, 3] = boxes[:, 3] + boxes[:, 1]
    return boxes


def overlay_boxes(frame, boxes, class_ids, label_map, color_map, line_thickness=None):
    tl = (
        line_thickness or round(0.0005 * (frame.shape[0] + frame.shape[1]) / 2) + 1
    )  # line/font thickness

    for class_id, (x1, y1, x2, y2) in zip(class_ids, boxes.astype(np.int)):
        color = color_map[class_id]
        label = label_map[class_id]
        cv2.rectangle(frame, (x1, y1), (x2, y2), color, tl, cv2.LINE_AA)
        tf = max(tl - 1, 1)  # font thickness
        t_size = cv2.getTextSize(label, 0, fontScale=tl / 3, thickness=tf)[0]
        x3, y3 = x1 + t_size[0], y1 - t_size[1] - 3
        cv2.rectangle(frame, (x1, y1), (x3, y3), color, -1, cv2.LINE_AA)  # filled
        cv2.putText(
            frame,
            label,
            (x1, y1 - 2),
            0,
            tl / 3,
            [225, 255, 255],
            thickness=tf,
            lineType=cv2.LINE_AA,
        )
