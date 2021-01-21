import torch
import regex
import tqdm


class PythonPredictor:
    def __init__(self, config):
        roberta = torch.hub.load("pytorch/fairseq", "roberta.large", force_reload=True)
        roberta.eval()
        device = "cuda" if torch.cuda.is_available() else "cpu"
        print(f"using device: {device}")
        roberta.to(device)

        self.model = roberta

    def predict(self, payload):
        predictions = self.model.fill_mask(payload["text"] + " <mask>", topk=5)
        return [prediction[0] for prediction in predictions]
