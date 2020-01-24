
import boto3
import json
import re
import base64
import numpy as np
import cv2
import keras
import tensorflow as tf
# import keras.backend.tensorflow_backend as tb
from os.path import split
# from keras.models import load_model
from utils.utils import get_yolo_boxes

class PythonPredictor():
    def __init__(self, config):
        bucket, key = re.match("s3://(.+?)/(.+)", config["model"]).groups()
        _, model_name = split(key)

        # self.session = tf.compat.v1.Session()
        # self.graph = tf.compat.v1.get_default_graph()
        # keras.backend.tensorflow_backend._SYMBOLIC_SCOPE.value = True

        s3 = boto3.client("s3")
        s3.download_file(bucket, key, model_name)
        self.model = keras.models.load_model(model_name)

        with open(config['model_config']) as json_file:
            data = json.load(json_file)
        for key in data:
            setattr(self, key, data[key])


    def predict(self, payload):
        img = payload["img"]
        img = base64.b64decode(img)
        jpg_as_np = np.frombuffer(img, dtype=np.uint8)
        image = cv2.imdecode(jpg_as_np, flags=cv2.IMREAD_COLOR)

        # with self.graph.as_default():
        #     with self.session.as_default():
        boxes = get_yolo_boxes(self.model, [image], self.net_h, self.net_w,
        self.anchors, self.obj_thresh, self.nms_thresh)[0]
        no_license_plates = len(boxes)

        return no_license_plates