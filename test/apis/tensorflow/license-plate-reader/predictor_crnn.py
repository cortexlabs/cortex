import cv2
import numpy as np
import keras_ocr
import base64
import pickle
import tensorflow as tf


class PythonPredictor:
    def __init__(self, config):
        # limit memory usage on each process
        for gpu in tf.config.list_physical_devices("GPU"):
            tf.config.experimental.set_memory_growth(gpu, True)

        # keras-ocr will automatically download pretrained
        # weights for the detector and recognizer.
        self.pipeline = keras_ocr.pipeline.Pipeline()

    def predict(self, payload):
        # preprocess the images w/ license plates (LPs)
        imgs = payload["imgs"]
        imgs = base64.b64decode(imgs.encode("utf-8"))
        jpgs_as_np = pickle.loads(imgs)
        images = [cv2.imdecode(jpg_as_np, flags=cv2.IMREAD_COLOR) for jpg_as_np in jpgs_as_np]

        # run batch inference
        try:
            prediction_groups = self.pipeline.recognize(images)
        except ValueError:
            # exception can occur when the images are too small
            prediction_groups = []

        image_list = []
        for img_predictions in prediction_groups:
            boxes_per_image = []
            for predictions in img_predictions:
                boxes_per_image.append([predictions[0], predictions[1].tolist()])
            image_list.append(boxes_per_image)

        lps = {"license-plates": image_list}

        return lps
