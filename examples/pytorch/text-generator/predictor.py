# WARNING: you are on the master branch, please refer to the examples on the branch that matches your `cortex version`

import torch
from transformers import GPT2Tokenizer, GPT2LMHeadModel


class PythonPredictor:
    def __init__(self, config):
        self.device = "cuda" if torch.cuda.is_available() else "cpu"
        print(f"using device: {self.device}")
        self.tokenizer = GPT2Tokenizer.from_pretrained("gpt2")
        self.model = GPT2LMHeadModel.from_pretrained("gpt2").to(self.device)

    def predict(self, payload, query_params):
        text = payload["text"]
        new_words_to_generate = int(query_params.get("words", 20))
        total_words = len(text.split(" ")) + new_words_to_generate

        tokens = self.tokenizer.encode(text, return_tensors="pt").to(self.device)
        prediction = self.model.generate(tokens, max_length=total_words, do_sample=True)
        return self.tokenizer.decode(prediction[0])
