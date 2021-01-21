import mlflow.sklearn


class PythonPredictor:
    def __init__(self, config, python_client):
        self.client = python_client

    def load_model(self, model_path):
        return mlflow.sklearn.load_model(model_path)

    def predict(self, payload, query_params):
        model_name = "mpg-estimator"
        model_version = query_params.get("version", "latest")

        model = self.client.get_model(model_name, model_version)
        model_input = [
            payload["cylinders"],
            payload["displacement"],
            payload["horsepower"],
            payload["weight"],
            payload["acceleration"],
        ]
        result = model.predict([model_input]).item()

        return {"prediction": result, "model": {"name": model_name, "version": model_version}}
