# WARNING: you are on the master branch, please refer to the examples on the branch that matches your `cortex version`

import boto3
import os

labels = ["setosa", "versicolor", "virginica"]


class ONNXPredictor:
    def __init__(self, onnx_client, config, job_spec):
        self.client = onnx_client
        # self.s3 = boto3.client("s3")
        # self.job_id = job_spec["job_id"]
        # self.bucket = config["bucket"]
        # self.prefix = os.path.join(config["prefix"], self.job_id)

    def predict(self, payload):
        model_input = [
            [
                payload["sepal_length"],
                payload["sepal_width"],
                payload["petal_length"],
                payload["petal_width"],
            ],
            [
                payload["sepal_length"],
                payload["sepal_width"],
                payload["petal_length"],
                payload["petal_width"],
            ],
        ]

        prediction = self.client.predict(model_input)
        print(prediction)
        predicted_class_id = prediction[0][0]
        return labels[predicted_class_id]
