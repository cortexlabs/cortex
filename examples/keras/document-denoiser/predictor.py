# WARNING: you are on the master branch, please refer to the examples on the branch that matches your `cortex version`

import boto3, base64, cv2, re, os, json
import numpy as np
from tensorflow.keras.models import load_model


def image_to_png_nparray(image):
    """
    Convert numpy image to jpeg numpy vector.
    """
    is_success, im_buf_arr = cv2.imencode(".png", image)
    return im_buf_arr


def image_to_png_bytes(image):
    """
    Convert numpy image to bytes-encoded png image.
    """
    buf = image_to_png_nparray(image)
    byte_im = buf.tobytes()
    return byte_im


def image_from_bytes(byte_im):
    """
    Convert image from bytes representation to numpy array.
    """
    nparr = np.frombuffer(byte_im, np.uint8)
    img_np = cv2.imdecode(nparr, cv2.IMREAD_GRAYSCALE)
    return img_np


class PythonPredictor:
    def __init__(self, config):
        # download the model
        model_path = config["model"]
        model_name = "model.h5"
        bucket, key = re.match("s3://(.+?)/(.+)", model_path).groups()
        s3 = boto3.client("s3")
        s3.download_file(bucket, os.path.join(key, model_name), model_name)

        # load the model
        self.model = load_model(model_name)

        # resize shape (width, height)
        self.resize_shape = tuple(config["resize_shape"])

    def predict(self, payload):
        # decode image
        img = payload["img"]
        img = base64.b64decode(img.encode("utf-8"))

        # convert bytes representation to image
        image = image_from_bytes(img)
        image = cv2.resize(image, self.resize_shape)

        # prediction
        pred = self.make_prediction(image)

        # image represented in bytes
        byte_im = image_to_png_bytes(pred)

        # encode image
        image_enc = base64.b64encode(byte_im).decode("utf-8")
        image_dump = json.dumps({"img_pred": image_enc})

        return image_dump

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
