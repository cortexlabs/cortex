# WARNING: you are on the master branch, please refer to the examples on the branch that matches your `cortex version`

import wget
import torch
from transformers import GPT2Tokenizer, GPT2LMHeadModel, GPT2Config
import generator


class Predictor:
    def __init__(self, config):
        medium_config = GPT2Config(n_embd=1024, n_layer=24, n_head=16)
        model = GPT2LMHeadModel(medium_config)
        wget.download(
            "https://convaisharables.blob.core.windows.net/lsp/multiref/medium_ft.pkl",
            "medium_ft.pkl",
        )

        weights = torch.load("medium_ft.pkl")
        weights["lm_head.weight"] = weights["lm_head.decoder.weight"]
        weights.pop("lm_head.decoder.weight", None)

        model.load_state_dict(weights)
        model.eval()
        model.to(config["device"])

        self.device = config["device"]
        self.model = model
        self.tokenizer = GPT2Tokenizer.from_pretrained("gpt2")

    def predict(self, payload):
        conditioned_tokens = self.tokenizer.encode(payload["text"]) + [generator.END_OF_TEXT]
        prediction = generator.generate(self.model, conditioned_tokens, self.device)
        return self.tokenizer.decode(prediction)
