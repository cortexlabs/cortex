# WARNING: you are on the master branch, please refer to the examples on the branch that matches your `cortex version`

import torch
from transformers import GPT2Tokenizer, GPT2LMHeadModel


class PythonPredictor:
    def __init__(self, config):
        self.tokenizer = GPT2Tokenizer.from_pretrained("gpt2")
        self.model = GPT2LMHeadModel.from_pretrained("gpt2")

    def predict(self, payload, query_params, headers):
        tokens = self.tokenizer.encode(payload["text"])
        prediction = self.model(torch.tensor([tokens]))
        token = torch.argmax(prediction[0][0, -1, :]).item()
        return self.tokenizer.decode(tokens + [token])
