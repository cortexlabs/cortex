from summarizer import Summarizer


model = Summarizer()


def predict(sample, metadata):
    return model(sample["text"])
