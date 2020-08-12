# WARNING: you are on the master branch, please refer to the examples on the branch that matches your `cortex version`

import requests
import numpy as np
from PIL import Image
from io import BytesIO
import json
import os
import re
import boto3
import tensorflow as tf


class TensorFlowPredictor:
    def __init__(self, tensorflow_client, config, job_spec):
        self.client = tensorflow_client
        self.labels = requests.get(
            "https://storage.googleapis.com/download.tensorflow.org/data/ImageNetLabels.txt"
        ).text.split("\n")[1:]

        if len(config["dest_s3_dir"]) == 0:
            raise Exception("'dest_s3_dir' field was not provided in job submission")

        self.s3 = boto3.client("s3")

        self.bucket, self.key = re.match("s3://(.+?)/(.+)", config["dest_s3_dir"]).groups()
        self.key = os.path.join(self.key, job_spec["job_id"])

    def predict(self, payload, batch_id):
        arr_list = []

        for img_url in payload:
            image = requests.get(img_url).content
            decoded_image = np.asarray(Image.open(BytesIO(image)), dtype=np.float32) / 255
            resized_image = tf.image.resize(
                decoded_image, [224, 224], method=tf.image.ResizeMethod.BILINEAR
            )
            arr_list.append(resized_image)

        model_input = {"images": np.stack(arr_list, axis=0)}
        predictions = self.client.predict(model_input)

        reshaped_predictions = np.reshape(np.array(predictions["classes"]), [-1, len(self.labels)])
        predicted_classes = np.argmax(reshaped_predictions, axis=1)

        results = [
            {"url": payload[i], "class": self.labels[class_idx]}
            for i, class_idx in enumerate(predicted_classes)
        ]

        json_output = json.dumps(results)

        self.s3.put_object(Bucket=self.bucket, Key=f"{self.key}/{batch_id}.json", Body=json_output)
