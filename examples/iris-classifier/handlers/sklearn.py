import numpy as np
import boto3
import json


labels = ["iris-setosa", "iris-versicolor", "iris-virginica"]

s3 = boto3.client("s3")
scalars_obj = s3.get_object(Bucket="cortex-examples", Key="iris/scalars.json")
scalars = json.loads(scalars_obj["Body"].read().decode("utf-8"))
print("downloaded scalars: {}".format(scalars))


def pre_inference(sample, metadata):
    x = np.array(
        [
            sample["sepal_length"],
            sample["sepal_width"],
            sample["petal_length"],
            sample["petal_width"],
        ]
    )
    return (x - scalars["mean"]) / scalars["stddev"]


def post_inference(prediction, metadata):
    predicted_class_id = prediction[0][0]
    return labels[predicted_class_id]
