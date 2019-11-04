import boto3
import re
import numpy as np
from joblib import load


model = None


def init(metadata):
    global model
    s3 = boto3.client("s3")
    bucket, key = re.match(r"s3:\/\/(.+?)\/(.+)", metadata["model"]).groups()
    s3.download_file(bucket, key, "linreg.joblib")
    model = load("linreg.joblib")


def predict(sample, metadata):
    input_array = [
        sample["cylinders"],
        sample["displacement"],
        sample["horsepower"],
        sample["weight"],
        sample["acceleration"],
    ]

    result = model.predict([input_array])
    return np.asscalar(result)
