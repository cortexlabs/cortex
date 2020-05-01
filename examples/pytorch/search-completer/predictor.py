# WARNING: you are on the master branch, please refer to the examples on the branch that matches your `cortex version`

import torch
import regex
import tqdm


class PythonPredictor:
    def __init__(self, config):
        roberta = torch.hub.load("pytorch/fairseq", "roberta.large")
        roberta.eval()
        device = "cuda" if torch.cuda.is_available() else "cpu"
        roberta.to(device)

        self.model = roberta

    def predict(self, payload):
        predictions = self.model.fill_mask(payload["text"] + " <mask>", topk=5)
        return [prediction[0] for prediction in predictions]
