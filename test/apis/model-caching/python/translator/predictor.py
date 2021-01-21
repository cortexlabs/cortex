from transformers import MarianMTModel, MarianTokenizer, pipeline
import torch


class PythonPredictor:
    def __init__(self, config, python_client):
        self.client = python_client
        self.device = torch.cuda.current_device() if torch.cuda.is_available() else -1

    def load_model(self, model_path):
        return MarianMTModel.from_pretrained(model_path, local_files_only=True)

    def predict(self, payload):
        model_name = "opus-mt-" + payload["source_language"] + "-" + payload["destination_language"]
        tokenizer_path = "Helsinki-NLP/" + model_name
        model = self.client.get_model(model_name)
        tokenizer = MarianTokenizer.from_pretrained(tokenizer_path)

        inf_pipeline = pipeline(
            "text2text-generation", model=model, tokenizer=tokenizer, device=self.device
        )
        result = inf_pipeline(payload["text"])

        return result[0]
