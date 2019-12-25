# this is an example for cortex release 0.12 and may not deploy correctly on other releases of cortex your `cortex version`

from encoder import get_encoder


class TensorFlowPredictor:
    def __init__(self, tensorflow_client, config):
        self.client = tensorflow_client
        self.encoder = get_encoder()

    def predict(self, payload):
        model_input = {"context": [self.encoder.encode(payload["text"])]}
        prediction = self.client.predict(model_input)
        return self.encoder.decode(prediction["sample"])
