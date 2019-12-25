# this is an example for cortex release 0.12 and may not deploy correctly on other releases of cortex your `cortex version`

import wget
import fasttext


class PythonPredictor:
    def __init__(self, config):
        wget.download(
            "https://dl.fbaipublicfiles.com/fasttext/supervised-models/lid.176.bin", "model"
        )

        self.model = fasttext.load_model("model")

    def predict(self, payload):
        prediction = self.model.predict(payload["text"])
        language = prediction[0][0][-2:]
        return language
