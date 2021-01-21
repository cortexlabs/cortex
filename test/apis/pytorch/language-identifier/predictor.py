import wget
import fasttext


class PythonPredictor:
    def __init__(self, config):
        wget.download(
            "https://dl.fbaipublicfiles.com/fasttext/supervised-models/lid.176.bin", "/tmp/model"
        )

        self.model = fasttext.load_model("/tmp/model")

    def predict(self, payload):
        prediction = self.model.predict(payload["text"])
        language = prediction[0][0][-2:]
        return language
