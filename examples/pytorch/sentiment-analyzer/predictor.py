# WARNING: you are on the master branch; please refer to examples on the branch corresponding to your `cortex version` (e.g. for version 0.21.*, run `git checkout -b 0.21` or switch to the `0.21` branch on GitHub)

import torch
from transformers import pipeline


class PythonPredictor:
    def __init__(self, config):
        device = 0 if torch.cuda.is_available() else -1
        print(f"using device: {'cuda' if device == 0 else 'cpu'}")

        self.analyzer = pipeline(task="sentiment-analysis", device=device)

    def predict(self, payload):
        return self.analyzer(payload["text"])[0]
