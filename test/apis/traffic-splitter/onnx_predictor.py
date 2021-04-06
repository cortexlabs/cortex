import os
import onnxruntime as rt
import numpy as np

labels = ["setosa", "versicolor", "virginica"]


class PythonPredictor:
    def __init__(self, python_client, config):
        self.client = python_client

    def predict(self, payload):
        session = self.client.get_model()

        input_dict = {
            "input": np.array(
                [
                    payload["sepal_length"],
                    payload["sepal_width"],
                    payload["petal_length"],
                    payload["petal_width"],
                ],
                dtype="float32",
            ).reshape(1, 4),
        }
        prediction = session.run(["label"], input_dict)

        predicted_class_id = prediction[0][0]
        return labels[predicted_class_id]

    def load_model(self, model_path):
        """
        Load ONNX model from disk.
        """

        model_path = os.path.join(model_path, os.listdir(model_path)[0])
        return rt.InferenceSession(model_path)
