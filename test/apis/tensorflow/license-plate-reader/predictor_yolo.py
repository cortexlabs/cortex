import json
import base64
import numpy as np
import cv2
import pickle
import utils.utils as utils


class TensorFlowPredictor:
    def __init__(self, tensorflow_client, config):
        self.client = tensorflow_client

        with open(config["model_config"]) as json_file:
            data = json.load(json_file)
        for key in data:
            setattr(self, key, data[key])

    def predict(self, payload):
        # decode the payload
        img = payload["img"]
        img = base64.b64decode(img)
        jpg_as_np = np.frombuffer(img, dtype=np.uint8)
        image = cv2.imdecode(jpg_as_np, flags=cv2.IMREAD_COLOR)

        # detect the bounding boxes
        boxes = utils.get_yolo_boxes(
            self.client,
            image,
            self.net_h,
            self.net_w,
            self.anchors,
            self.obj_thresh,
            self.nms_thresh,
            len(self.labels),
        )

        # package the response
        response = {"boxes": []}
        for box in boxes:
            response["boxes"].append(
                [box.xmin, box.ymin, box.xmax, box.ymax, float(box.c), box.classes.tolist()]
            )

        return response
