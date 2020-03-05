# this is an example for cortex release 0.14 and may not deploy correctly on other releases of cortex

from allennlp.predictors.predictor import Predictor as AllenNLPPredictor


class PythonPredictor:
    def __init__(self, config):

        cuda_device = -1
        if config["device"] == "cuda":
            cuda_device = 0

        self.predictor = AllenNLPPredictor.from_path(
            "https://storage.googleapis.com/allennlp-public-models/bidaf-elmo-model-2018.11.30-charpad.tar.gz",
            cuda_device=cuda_device,
        )

    def predict(self, payload):
        prediction = self.predictor.predict(
            passage=payload["passage"], question=payload["question"]
        )
        return prediction["best_span_str"]
