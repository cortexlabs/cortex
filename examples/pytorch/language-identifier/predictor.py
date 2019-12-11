# WARNING: you are on the master branch, please refer to the examples on the branch that matches your `cortex version`

import wget
import fasttext


class Predictor:
    def __init__(self, config):
        wget.download(
            "https://dl.fbaipublicfiles.com/fasttext/supervised-models/lid.176.bin", "model"
        )

        self.model = fasttext.load_model("model")

    def predict(self, payload):
        prediction = self.model.predict(payload["text"])
        language = prediction[0][0][-2:]
        return language
