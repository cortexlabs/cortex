import torch
import transformers
from transformers import pipeline

class PythonPredictor:
    def __init__(self, config):
        self.summarizer = pipeline(task='summarization')

    def predict(self, payload):
        summary = self.summarizer(
            payload["text"],
            num_beams=4,
            length_penalty=2.0,
            max_length=142,
            no_repeat_ngram_size=3
        )
        return summary[0]['summary_text']