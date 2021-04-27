import mlflow.sklearn


class Handler:
    def __init__(self, config, model_client):
        self.client = model_client

    def load_model(self, model_path):
        return mlflow.sklearn.load_model(model_path)

    def handle_post(self, payload, query_params):
        model_version = query_params.get("version", "latest")

        model = self.client.get_model(model_version=model_version)
        model_input = [
            payload["cylinders"],
            payload["displacement"],
            payload["horsepower"],
            payload["weight"],
            payload["acceleration"],
        ]
        result = model.predict([model_input]).item()

        return {"prediction": result, "model": {"version": model_version}}
