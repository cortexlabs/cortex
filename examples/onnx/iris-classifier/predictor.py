# WARNING: you are on the master branch; please refer to examples on the branch corresponding to your `cortex version` (e.g. for version 0.19.*, run `git checkout -b 0.19` or switch to the `0.19` branch on GitHub)

labels = ["setosa", "versicolor", "virginica"]


class ONNXPredictor:
    def __init__(self, onnx_client, config):
        self.client = onnx_client

    def predict(self, payload):
        model_input = [
            payload["sepal_length"],
            payload["sepal_width"],
            payload["petal_length"],
            payload["petal_width"],
        ]

        prediction = self.client.predict(model_input)
        predicted_class_id = prediction[0][0]
        return labels[predicted_class_id]
