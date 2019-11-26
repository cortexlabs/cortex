import wget
import fasttext


wget.download("https://dl.fbaipublicfiles.com/fasttext/supervised-models/lid.176.bin", "model")
model = fasttext.load_model("model")


def predict(sample, metadata):
    prediction = model.predict(sample["text"])
    language = prediction[0][0][-2:]
    return language
