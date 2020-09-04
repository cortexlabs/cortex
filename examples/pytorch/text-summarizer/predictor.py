# WARNING: you are on the master branch; please refer to examples on the branch corresponding to your `cortex version` (e.g. for version 0.19.*, run `git checkout -b 0.19` or switch to the `0.19` branch on GitHub)

import torch
from transformers import pipeline


class PythonPredictor:
    def __init__(self, config):
        device = 0 if torch.cuda.is_available() else -1
        print(f"using device: {'cuda' if device == 0 else 'cpu'}")

        self.summarizer = pipeline(task="summarization", device=device)

    def predict(self, payload):
        summary = self.summarizer(
            payload["text"], num_beams=4, length_penalty=2.0, max_length=142, no_repeat_ngram_size=3
        )
        return summary[0]["summary_text"]
