import mlflow.sklearn
import numpy as np


model = mlflow.sklearn.load_model("s3://cortex-examples/sklearn/mpg-estimation/linreg")


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
