# WARNING: you are on the master branch, please refer to the examples on the branch that matches your `cortex version`

import ffmpeg
import youtube_dl
import uuid
from pathlib import Path
import numpy as np
import cv2
from starlette.responses import FileResponse
from typing import Iterable, Tuple
import os


def download_from_youtube(url: str) -> Path:
    target = f'{uuid.uuid1()}.mp4'
    ydl_opts = {'outtmpl': target, 'format': "worstvideo[vcodec=vp9][height>=480]"}
    with youtube_dl.YoutubeDL(ydl_opts) as ydl:
        ydl.download([url])
    # we need to glob in case youtube-dl adds suffix
    path, = Path().absolute().glob(f'{target}*')
    return path


def frame_reader(path: Path, size: Tuple[int, int]) -> Iterable[np.ndarray]:
    width, height = size
    # letterbox frames to fixed size
    process = (
        ffmpeg
        .input(path)
        .filter('scale', size=f'{width}:{height}', force_original_aspect_ratio='decrease')
        # Negative values for x and y center the padded video
        .filter('pad', height=height, width=width, x=-1, y=-1)
        .output('pipe:', format='rawvideo', pix_fmt='rgb24')
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
    def __init__(self, path, size):
        width, height = size
        self.process = (
            ffmpeg
            .input('pipe:', format='rawvideo', pix_fmt='rgb24', s=f'{width}x{height}')
            .output(path, pix_fmt='yuv420p')
            .overwrite_output()
            .run_async(pipe_stdin=True)
        )

    def write(self, frame):
        self.process.stdin.write(frame.astype(np.uint8).tobytes())


    def __del__(self):
        self.process.stdin.close()
        self.process.wait()


def nms(dets, scores, thresh):
    x1 = dets[:, 0]
    y1 = dets[:, 1]
    x2 = dets[:, 2]
    y2 = dets[:, 3]

    areas = (x2 - x1 + 1) * (y2 - y1 + 1)
    order = scores.argsort()[::-1] # get boxes with more ious first

    keep = []
    while order.size > 0:
        i = order[0] # pick maxmum iou box
        keep.append(i)
        xx1 = np.maximum(x1[i], x1[order[1:]])
        yy1 = np.maximum(y1[i], y1[order[1:]])
        xx2 = np.minimum(x2[i], x2[order[1:]])
        yy2 = np.minimum(y2[i], y2[order[1:]])

        w = np.maximum(0.0, xx2 - xx1 + 1) # maximum width
        h = np.maximum(0.0, yy2 - yy1 + 1) # maxiumum height
        inter = w * h
        ovr = inter / (areas[i] + areas[order[1:]] - inter)

        inds = np.where(ovr <= thresh)[0]
        order = order[inds + 1]

    return keep



class ONNXPredictor:
    def __init__(self, onnx_client, config):
        self.client = onnx_client
        self.config = config

    def postprocess(self, output):
        boxes, obj_score, class_scores = np.split(output[0], [4, 5], axis=1)
        # convert boxes from x_center, y_center, w, h to xyxy
        boxes[:, 0] -= boxes[:, 2] / 2
        boxes[:, 1] -= boxes[:, 3] / 2
        boxes[:, 2] = boxes[:, 2] + boxes[:, 0]
        boxes[:, 3] = boxes[:, 3] + boxes[:, 1]
        class_id = class_scores.argmax(axis=1)
        cls_score = class_scores[np.arange(len(class_scores)), class_id]
        confidence = obj_score.squeeze(axis=1) * cls_score
        sel = confidence > self.config['confidence_threshold']
        boxes, class_id, confidence = boxes[sel], class_id[sel], confidence[sel]
        sel = nms(boxes, confidence, self.config['iou_threshold'])
        boxes, class_id, confidence = boxes[sel], class_id[sel], confidence[sel]
        return boxes, class_id, confidence

    def predict(self, payload):
        in_path = download_from_youtube(payload['url'])
        out_path = f'{uuid.uuid1()}.mp4'
        writer = FrameWriter(out_path, size=(416, 416))

        # TODO Get size from ONNX
        for frame in frame_reader(in_path, size=(416, 416)):
            x = (frame.astype(np.float32) / 255).transpose(2, 0, 1)
            # 4 output tensors, the last three are intermediate values and
            # not necessary for detection
            output, *_ = self.client.predict(x[None])
            boxes, class_id, confidence = self.postprocess(output)
            for x1, y1, x2, y2 in boxes:
                frame = cv2.rectangle(frame, (x1, y1), (x2, y2), (255, 0, 0), 1)
            writer.write(frame)

        del writer
        return FileResponse(out_path)