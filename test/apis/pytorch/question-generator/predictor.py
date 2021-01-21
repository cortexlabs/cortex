from transformers import AutoModelWithLMHead, AutoTokenizer
import spacy
import subprocess
import json


class PythonPredictor:
    def __init__(self, config):
        subprocess.call("python -m spacy download en_core_web_sm".split(" "))
        import en_core_web_sm

        self.tokenizer = AutoTokenizer.from_pretrained(
            "mrm8488/t5-base-finetuned-question-generation-ap"
        )
        self.model = AutoModelWithLMHead.from_pretrained(
            "mrm8488/t5-base-finetuned-question-generation-ap"
        )
        self.nlp = en_core_web_sm.load()

    def predict(self, payload):
        context = payload["context"]
        answer = payload["answer"]
        max_length = int(payload.get("max_length", 64))

        input_text = "answer: {}  context: {} </s>".format(answer, context)
        features = self.tokenizer([input_text], return_tensors="pt")

        output = self.model.generate(
            input_ids=features["input_ids"],
            attention_mask=features["attention_mask"],
            max_length=max_length,
        )

        return {"result": self.tokenizer.decode(output[0])}
