import wget
import torch
from transformers import GPT2Tokenizer, GPT2LMHeadModel, GPT2Config
import generator


class PythonPredictor:
    def __init__(self, config):
        medium_config = GPT2Config(n_embd=1024, n_layer=24, n_head=16)
        model = GPT2LMHeadModel(medium_config)
        wget.download(
            "https://convaisharables.blob.core.windows.net/lsp/multiref/medium_ft.pkl",
            "/tmp/medium_ft.pkl",
        )

        weights = torch.load("/tmp/medium_ft.pkl")
        weights["lm_head.weight"] = weights["lm_head.decoder.weight"]
        weights.pop("lm_head.decoder.weight", None)

        model.load_state_dict(weights)

        device = "cuda" if torch.cuda.is_available() else "cpu"
        print(f"using device: {device}")
        model.to(device)
        model.eval()

        self.device = device
        self.model = model
        self.tokenizer = GPT2Tokenizer.from_pretrained("gpt2")

    def predict(self, payload):
        conditioned_tokens = self.tokenizer.encode(payload["text"]) + [generator.END_OF_TEXT]
        prediction = generator.generate(self.model, conditioned_tokens, self.device)
        return self.tokenizer.decode(prediction)
