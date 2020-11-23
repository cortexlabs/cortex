# WARNING: you are on the master branch; please refer to examples on the branch corresponding to your `cortex version` (e.g. for version 0.22.*, run `git checkout -b 0.22` or switch to the `0.22` branch on GitHub)

from transformers import MarianMTModel, MarianTokenizer, pipeline


class PythonPredictor:
    def __init__(self, config, python_client):
        self.client = python_client

    def load_model(self, model_path):
        return MarianMTModel.from_pretrained(model_path, local_files_only=True)

    def predict(self, payload):
        model_name = "opus-mt-" + payload["source_language"] + "-" + payload["destination_language"]
        tokenizer_path = "Helsinki-NLP/" + model_name
        model = self.client.get_model(model_name)
        tokenizer = MarianTokenizer.from_pretrained(tokenizer_path)

        inf_pipeline = pipeline("text2text-generation", model=model, tokenizer=tokenizer)
        result = inf_pipeline(payload["text"])

        return result[0]
