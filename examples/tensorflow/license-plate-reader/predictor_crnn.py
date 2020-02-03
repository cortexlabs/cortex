
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
        print(len(jpgs_as_np), jpgs_as_np)
        print(type(jpgs_as_np[0]))
        print(jpgs_as_np[0].shape)
        images = [cv2.imdecode(jpg_as_np, flags=cv2.IMREAD_COLOR) for jpg_as_np in jpgs_as_np]
        
        # run batch inference
        prediction_groups = self.pipeline.recognize(images)
        lps = {"license-plates": []}
        for pred in prediction_groups:
            words = [x[0] for x in pred]
            lps["license-plates"].append(words)
            
        return lps