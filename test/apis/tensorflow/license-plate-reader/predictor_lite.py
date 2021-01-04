import boto3, base64, cv2, re, os, requests, json
import keras_ocr

from botocore import UNSIGNED
from botocore.client import Config
from tensorflow.keras.models import load_model
import utils.utils as utils
import utils.bbox as bbox_utils
import utils.preprocess as preprocess_utils


class PythonPredictor:
    def __init__(self, config):
        # download yolov3 model
        bucket, key = re.match("s3://(.+?)/(.+)", config["yolov3"]).groups()

        if os.environ.get("AWS_ACCESS_KEY_ID"):
            s3 = boto3.client("s3")  # client will use your credentials if available
        else:
            s3 = boto3.client("s3", config=Config(signature_version=UNSIGNED))  # anonymous client

        model_path = "/tmp/model.h5"
        s3.download_file(bucket, key, model_path)

        # load yolov3 model
        self.yolov3_model = load_model(model_path)

        # get configuration for yolov3 model
        with open(config["yolov3_model_config"]) as json_file:
            data = json.load(json_file)
        for key in data:
            setattr(self, key, data[key])
        self.box_confidence_score = 0.8

        # keras-ocr automatically downloads the pretrained
        # weights for the detector and recognizer
        self.recognition_model_pipeline = keras_ocr.pipeline.Pipeline()

    def predict(self, payload):
        # download image
        img_url = payload["url"]
        image = preprocess_utils.get_url_image(img_url)

        # detect the bounding boxes
        boxes = utils.get_yolo_boxes(
            self.yolov3_model,
            image,
            self.net_h,
            self.net_w,
            self.anchors,
            self.obj_thresh,
            self.nms_thresh,
            len(self.labels),
            tensorflow_model=False,
        )

        # purge bounding boxes with a low confidence score
        aux = []
        for b in boxes:
            label = -1
            for i in range(len(b.classes)):
                if b.classes[i] > self.box_confidence_score:
                    label = i
            if label >= 0:
                aux.append(b)
        boxes = aux
        del aux

        # if bounding boxes have been detected
        dec_words = []
        if len(boxes) > 0:
            # create set of images of the detected license plates
            lps = []
            for b in boxes:
                lp = image[b.ymin : b.ymax, b.xmin : b.xmax]
                lps.append(lp)

            # run batch inference
            try:
                prediction_groups = self.recognition_model_pipeline.recognize(lps)
            except ValueError:
                # exception can occur when the images are too small
                prediction_groups = []

            # process pipeline output
            image_list = []
            for img_predictions in prediction_groups:
                boxes_per_image = []
                for predictions in img_predictions:
                    boxes_per_image.append([predictions[0], predictions[1].tolist()])
                image_list.append(boxes_per_image)

            # reorder text within detected LPs based on horizontal position
            dec_lps = preprocess_utils.reorder_recognized_words(image_list)
            for dec_lp in dec_lps:
                dec_words.append([word[0] for word in dec_lp])

        # if there are no recognized LPs, then don't draw them
        if len(dec_words) == 0:
            dec_words = [[] for i in range(len(boxes))]

        # draw predictions as overlays on the source image
        draw_image = bbox_utils.draw_boxes(
            image,
            boxes,
            overlay_text=dec_words,
            labels=["LP"],
            obj_thresh=self.box_confidence_score,
        )

        # image represented in bytes
        byte_im = preprocess_utils.image_to_jpeg_bytes(draw_image)

        # encode image
        image_enc = base64.b64encode(byte_im).decode("utf-8")

        # image with draw boxes overlayed
        return image_enc
