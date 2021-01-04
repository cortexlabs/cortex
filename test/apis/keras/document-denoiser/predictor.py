import boto3, base64, cv2, re, os, requests
from botocore import UNSIGNED
from botocore.client import Config
import numpy as np
from tensorflow.keras.models import load_model


def get_url_image(url_image):
    """
    Get numpy image from URL image.
    """
    resp = requests.get(url_image, stream=True).raw
    image = np.asarray(bytearray(resp.read()), dtype="uint8")
    image = cv2.imdecode(image, cv2.IMREAD_GRAYSCALE)
    return image


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


class PythonPredictor:
    def __init__(self, config):
        # download the model
        bucket, key = re.match("s3://(.+?)/(.+)", config["model"]).groups()

        if os.environ.get("AWS_ACCESS_KEY_ID"):
            s3 = boto3.client("s3")  # client will use your credentials if available
        else:
            s3 = boto3.client("s3", config=Config(signature_version=UNSIGNED))  # anonymous client

        model_path = os.path.join("/tmp/model.h5")
        s3.download_file(bucket, key, model_path)

        # load the model
        self.model = load_model(model_path)

        # resize shape (width, height)
        self.resize_shape = tuple(config["resize_shape"])

    def predict(self, payload):
        # download image
        img_url = payload["url"]
        image = get_url_image(img_url)
        resized = cv2.resize(image, self.resize_shape)

        # prediction
        pred = self.make_prediction(resized)

        # image represented in bytes
        byte_im = image_to_png_bytes(pred)

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
