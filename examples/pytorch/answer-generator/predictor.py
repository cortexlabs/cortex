import wget
import torch
from transformers import GPT2Tokenizer, GPT2LMHeadModel, GPT2Config
import generator


medium_config = GPT2Config(n_embd=1024, n_layer=24, n_head=16)
model = GPT2LMHeadModel(medium_config)
tokenizer = GPT2Tokenizer.from_pretrained("gpt2")


def init(model_path, metadata):
    wget.download(
        "https://convaisharables.blob.core.windows.net/lsp/multiref/medium_ft.pkl", "medium_ft.pkl"
    )

    weights = torch.load("medium_ft.pkl")
    weights["lm_head.weight"] = weights["lm_head.decoder.weight"]
    weights.pop("lm_head.decoder.weight", None)

    model.load_state_dict(weights)
    model.eval()
    model.to(metadata["device"])


def predict(payload, metadata):
    conditioned_tokens = tokenizer.encode(payload["text"]) + [generator.END_OF_TEXT]
    prediction = generator.generate(model, conditioned_tokens, metadata["device"])
    return tokenizer.decode(prediction)
