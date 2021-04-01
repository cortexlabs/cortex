import os
import onnxruntime as rt
import numpy as np

labels = ["setosa", "versicolor", "virginica"]

class PythonPredictor:
    def __init__(self, python_client, config):
        self.client = python_client

    def predict(self, payload):
        model = self.client.get_model()
        session = model["session"]
        input_name = model["input_name"]
        output_name = model["input_name"]

        input_dict = {
            input_name: np.array([
                payload["sepal_length"],
                payload["sepal_width"],
                payload["petal_length"],
                payload["petal_width"],
            ], dtype="float32").reshape(1, 4),
        }
        prediction = session.run([output_name], input_dict)

        predicted_class_id = prediction[0][0]
        return labels[predicted_class_id]

    def load_model(self, model_path):
        """
        Load ONNX model from disk.
        """

        model_path = os.path.join(model_path, os.listdir(model_path)[0])
        session = rt.InferenceSession(model_path)
        return {
            "session": session,
            "input_name": session.get_inputs()[0].name,
            "output_name": session.get_outputs()[0].name,
        }
