labels = ["setosa", "versicolor", "virginica"]


class Handler:
    def __init__(self, tensorflow_client, config):
        self.client = tensorflow_client

    def handle_async(self, payload):
        prediction = self.client.predict(payload)
        predicted_class_id = int(prediction["class_ids"][0])
        return {"label": labels[predicted_class_id]}
