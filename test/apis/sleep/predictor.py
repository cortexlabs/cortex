import time


class PythonPredictor:
    def __init__(self, config):
        pass

    def predict(self, payload, query_params):
        time.sleep(float(query_params.get("sleep", 0)))
        return "ok"
