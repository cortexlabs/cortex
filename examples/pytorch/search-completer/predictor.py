import torch
import regex
import tqdm


class Predictor:
    def __init__(self, metadata):
        roberta = torch.hub.load("pytorch/fairseq", "roberta.large")
        roberta.eval()
        roberta.cuda()

        self.model = roberta

    def predict(self, payload):
        predictions = self.model.fill_mask(payload["text"] + " <mask>", topk=5)
        return [prediction[0] for prediction in predictions]
