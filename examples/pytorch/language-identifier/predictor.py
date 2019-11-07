import wget
import fasttext

wget.download(
    "https://dl.fbaipublicfiles.com/fasttext/supervised-models/lid.176.bin", "lid.176.bin"
)
model = fasttext.load_model("lid.176.bin")


def predict(sample, metadata):
    prediction = model.predict(sample["text"])
    language = prediction[0][0][-2:]
    return language
