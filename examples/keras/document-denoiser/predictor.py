# WARNING: you are on the master branch, please refer to the examples on the branch that matches your `cortex version`

import boto3, base64, cv2, re, os, requests
from botocore import UNSIGNED
from botocore.client import Config
import numpy as np
from tensorflow.keras.models import load_model
import util


class PythonPredictor:
    def __init__(self, config):
        # download the model
        bucket, key = re.match("s3://(.+?)/(.+)", config["model"]).groups()

        3 = boto3.client("s3")  # client will use your credentials if available

        model_path = os.path.join("/tmp/model.h5")
        s3.download_file(bucket, key, model_path)

        # load the model
        self.model = load_model(model_path)

        # resize shape (width, height)
        self.resize_shape = tuple(config["resize_shape"])

    def predict(self, payload):
        # download image
        img_url = payload["url"]
        image = util.get_url_image(img_url)
        resized = cv2.resize(image, self.resize_shape)

        # prediction
        pred = self.make_prediction(resized)

        # image represented in bytes
        byte_im = util.image_to_png_bytes(pred)

        # encode image
        image_enc = base64.b64encode(byte_im).decode("utf-8")

        return image_enc

    def make_prediction(self, img):
        """
        Make prediction on image.
        """
        processed = img / 255.0
        processed = np.expand_dims(processed, 0)
        processed = np.expand_dims(processed, 3)
        pred = self.model.predict(processed)
        pred = np.squeeze(pred, 3)
        pred = np.squeeze(pred, 0)
        out_img = pred * 255
        out_img[out_img > 255.0] = 255.0
        out_img = out_img.astype(np.uint8)
        return out_img
