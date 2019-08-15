import numpy as np
import boto3
import json

from cortex.lib.log import get_logger

logger = get_logger()
s3 = boto3.client("s3")

iris_labels = ["Iris-setosa", "Iris-versicolor", "Iris-virginica"]

scalars_obj = s3.get_object(Bucket="cortex-examples", Key="iris/scalars.json")
scalars = json.loads(scalars_obj["Body"].read().decode("utf-8"))
logger.info("downloaded scalars: {}".format(scalars))


def pre_inference(sample, metadata):
    x = np.array(
        [
            sample["sepal_length"],
            sample["sepal_width"],
            sample["petal_length"],
            sample["petal_width"],
        ]
    )
    return ((x - scalars["mean"]) / scalars["stddev"]).astype(np.float32)


def post_inference(prediction, metadata):
    predicted_class_id = prediction[0][0]
    return {"class_label": iris_labels[predicted_class_id], "class_index": predicted_class_id}
