from joblib import load
import boto3
import numpy as np


model = None


def init(metadata):
    global model
    s3 = boto3.client("s3")
    s3.download_file(metadata["bucket"], metadata["key"], "mpg.joblib")
    model = load("mpg.joblib")


def predict(sample, metadata):
    arr = [
        sample["cylinders"],
        sample["displacement"],
        sample["horsepower"],
        sample["weight"],
        sample["acceleration"],
    ]
    result = model.predict([arr])
    return np.asscalar(result)
