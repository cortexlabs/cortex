# WARNING: you are on the master branch, please refer to the examples on the branch that matches your `cortex version`

from encoder import get_encoder


class TensorFlowPredictor:
    def __init__(self, tf_client, config):
        self._tf_client = tf_client
        self._encoder = get_encoder()

    def predict(self, payload):
        model_input = {"context": [self._encoder.encode(payload["text"])]}
        prediction = self._tf_client.predict(model_input)
        return self._encoder.decode(prediction["sample"])
