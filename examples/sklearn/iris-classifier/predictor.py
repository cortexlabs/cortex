import boto3
import pickle
import numpy


# Download the model

object = boto3.client("s3").get_object(
    Bucket="cortex-examples", Key="sklearn/iris-classifier/model.pkl"
)
model = pickle.loads(object["Body"].read())
labels = ["iris-setosa", "iris-versicolor", "iris-virginica"]


def predict(sample, metadata):
    input_array = numpy.array(
        [
            sample["sepal_length"],
            sample["sepal_width"],
            sample["petal_length"],
            sample["petal_width"],
        ]
    )

    label_id = model.predict([input_array])[0]
    return labels[label_id]
