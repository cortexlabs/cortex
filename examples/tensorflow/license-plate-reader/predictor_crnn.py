import cv2
import numpy as np
import keras_ocr
import base64
import pickle


class PythonPredictor:
    def __init__(self, config):
        # keras-ocr will automatically download pretrained
        # weights for the detector and recognizer.
        self.pipeline = keras_ocr.pipeline.Pipeline()

    def predict(self, payload):
        # preprocess the images w/ license plates (LPs)
        imgs = payload["imgs"]
        imgs = base64.b64decode(imgs.encode("utf-8"))
        jpgs_as_np = pickle.loads(imgs)
        images = [
            cv2.imdecode(jpg_as_np, flags=cv2.IMREAD_COLOR) for jpg_as_np in jpgs_as_np
        ]

        # run batch inference
        prediction_groups = self.pipeline.recognize(images)
        for img_predictions in prediction_groups:
            for predictions in img_predictions:
                predictions = tuple([predictions[0], predictions[1].tolist()])
        lps = {"license-plates": prediction_groups}

        return lps
