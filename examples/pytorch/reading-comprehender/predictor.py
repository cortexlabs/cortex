from allennlp.predictors.predictor import Predictor


predictor = Predictor.from_path(
    "https://storage.googleapis.com/allennlp-public-models/bidaf-elmo-model-2018.11.30-charpad.tar.gz"
)


def predict(payload, metadata):
    prediction = predictor.predict(passage=payload["passage"], question=payload["question"])
    return prediction["best_span_str"]
