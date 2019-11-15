from allennlp.predictors.predictor import Predictor


predictor = Predictor.from_path(
    "https://storage.googleapis.com/allennlp-public-models/bidaf-elmo-model-2018.11.30-charpad.tar.gz"
)


def predict(sample, metadata):
    prediction = predictor.predict(passage=sample["passage"], question=sample["question"])
    return prediction["best_span_str"]
