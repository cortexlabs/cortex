import json
import os
import io
import uuid
import utils

import numpy as np
from matplotlib import pyplot as plt

from starlette.responses import StreamingResponse


class ONNXPredictor:
    def __init__(self, onnx_client, config):
        self.client = onnx_client
        # Get the input shape from the ONNX runtime
        (signature,) = onnx_client.get_model()["input_signatures"].values()
        _, _, height, width = signature["shape"]
        self.input_size = (width, height)
        self.config = config
        with open("labels.json") as buf:
            self.labels = json.load(buf)
        color_map = plt.cm.tab20(np.linspace(0, 20, len(self.labels)))
        self.color_map = [tuple(map(int, colors)) for colors in 255 * color_map]

    def postprocess(self, output):
        boxes, obj_score, class_scores = np.split(output[0], [4, 5], axis=1)
        boxes = utils.boxes_yolo_to_xyxy(boxes)

        # get the class-prediction & class confidences
        class_id = class_scores.argmax(axis=1)
        cls_score = class_scores[np.arange(len(class_scores)), class_id]

        confidence = obj_score.squeeze(axis=1) * cls_score
        sel = confidence > self.config["confidence_threshold"]
        boxes, class_id, confidence = boxes[sel], class_id[sel], confidence[sel]
        sel = utils.nms(boxes, confidence, self.config["iou_threshold"])
        boxes, class_id, confidence = boxes[sel], class_id[sel], confidence[sel]
        return boxes, class_id, confidence

    def predict(self, payload):
        # download YT video
        in_path = utils.download_from_youtube(payload["url"], self.input_size[1])
        out_path = f"{uuid.uuid1()}.mp4"

        # run predictions
        with utils.FrameWriter(out_path, size=self.input_size) as writer:
            for frame in utils.frame_reader(in_path, size=self.input_size):
                x = (frame.astype(np.float32) / 255).transpose(2, 0, 1)
                # 4 output tensors, the last three are intermediate values and
                # not necessary for detection
                output, *_ = self.client.predict(x[None])
                boxes, class_ids, confidence = self.postprocess(output)
                utils.overlay_boxes(frame, boxes, class_ids, self.labels, self.color_map)
                writer.write(frame)

        with open(out_path, "rb") as f:
            output_buf = io.BytesIO(f.read())

        os.remove(in_path)
        os.remove(out_path)

        return StreamingResponse(output_buf, media_type="video/mp4")
