
import boto3
import re
import string
from os.path import split

from crnn import CRNN_STN
import keras.backend as K
import cv2
import numpy as np

def pad_image(img, img_size, nb_channels):
    img_reshape = cv2.resize(img, (int(img_size[1] / img.shape[0] * img.shape[1]), img_size[1]))
    if nb_channels == 1:
        padding = np.zeros((img_size[1], img_size[0] - int(img_size[1] / img.shape[0] * img.shape[1])), dtype=np.int32)
    else:
        padding = np.zeros((img_size[1], img_size[0] - int(img_size[1] / img.shape[0] * img.shape[1]), nb_channels), dtype=np.int32)
    img = np.concatenate([img_reshape, padding], axis=1)
    return img

def resize_image(img, img_size):
    img = cv2.resize(img, img_size, interpolation=cv2.INTER_CUBIC)
    img = np.asarray(img)
    return img

class PythonPredictor:
    def __init__(self, config):
        bucket, key = re.match("s3://(.+?)/(.+)", config["model"]).groups()
        _, model_name = split(key)

        s3 = boto3.client("s3")
        s3.download_file(bucket, key, model_name)

        self.get_default_model_params()
        _, self.model = CRNN_STN(self.cfg)
        self.model.load_weights(model_name)

    def predict(self, payload):
        img = payload["img"]
        img = base64.b64decode(img)
        jpg_as_np = np.frombuffer(img, dtype=np.uint8)
        image = cv2.imdecode(jpg_as_np, flags=cv2.IMREAD_GRAYSCALE)
        image = self.preprocess_image(image)

        pred = self.model.predict_on_batch(image)
        print(pred)

    def get_default_model_params(self):
        ModelConfig = namedtuple("ModelConfig", [
            "characters", "label_len", "nb_channels", "width", "height", 
            "conv_filter_size", "lstm_nb_units", "timesteps", "dropout_rate"
        ])
        cfg = ModelConfig(
            "0123456789" + string.ascii_lowercase+"-",
            16, 1, 200, 31, [64, 128, 256, 256, 512, 512, 512],
            [128, 128], 50, 0.25)

        self.cfg = cfg

    def preprocess_image(self, img):
        if img.shape[1] / img.shape[0] < 6.4:
            img = pad_image(img, (self.cfg.width, self.cfg.height), self.cfg.nb_channels)
        else:
            img = resize_image(img, (self.cfg.width, self.cfg.height))
        if self.cfg.nb_channels == 1:
            img = img.transpose([1, 0])
        else:
            img = img.transpose([1, 0, 2])
        img = np.flip(img, 1)
        img = img / 255.0
        if self.cfg.nb_channels == 1:
            img = img[:, :, np.newaxis]
        return img

    def predict_text(self, img):
        y_pred = self.model.predict(img[np.newaxis, :, :, :])
        shape = y_pred[:, 2:, :].shape
        ctc_decode = K.ctc_decode(y_pred[:, 2:, :], input_length=np.ones(shape[0])*shape[1])[0][0]
        ctc_out = K.get_value(ctc_decode)[:, :self.cfg.label_len]
        result_str = ''.join([cfg.characters[c] for c in ctc_out[0]])
        result_str = result_str.replace('-', '')
        return result_str