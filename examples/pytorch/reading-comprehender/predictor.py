# WARNING: you are on the master branch, please refer to the examples on the branch that matches your `cortex version`

from allennlp.predictors.predictor import Predictor as AllenNLPPredictor


class PythonPredictor:
    def __init__(self, config):
        self.predictor = AllenNLPPredictor.from_path(
            "https://storage.googleapis.com/allennlp-public-models/bidaf-elmo-model-2018.11.30-charpad.tar.gz"
        )

    def predict(self, payload):
        prediction = self.predictor.predict(
            passage=payload["passage"], question=payload["question"]
        )
        return prediction["best_span_str"]
