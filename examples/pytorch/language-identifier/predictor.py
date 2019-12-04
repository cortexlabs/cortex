import wget
import fasttext


wget.download("https://dl.fbaipublicfiles.com/fasttext/supervised-models/lid.176.bin", "model")
model = fasttext.load_model("model")


def predict(payload, metadata):
    prediction = model.predict(payload["text"])
    language = prediction[0][0][-2:]
    return language
