from summarizer import Summarizer


model = Summarizer()


def predict(payload, metadata):
    return model(payload["text"])
