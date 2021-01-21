import torch
from transformers import pipeline
from starlette.responses import JSONResponse


class PythonPredictor:
    def __init__(self, config):
        device = 0 if torch.cuda.is_available() else -1
        print(f"using device: {'cuda' if device == 0 else 'cpu'}")

        self.analyzer = pipeline(task="sentiment-analysis", device=device)
        self.summarizer = pipeline(task="summarization", device=device)

    def predict(self, query_params, payload):
        model_name = query_params.get("model")

        if model_name == "sentiment":
            return self.analyzer(payload["text"])[0]
        elif model_name == "summarizer":
            summary = self.summarizer(payload["text"])
            return summary[0]["summary_text"]
        else:
            return JSONResponse({"error": f"unknown model: {model_name}"}, status_code=400)
