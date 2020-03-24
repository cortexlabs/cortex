import torch
import transformers
from transformers import BartTokenizer, BartForConditionalGeneration

class PythonPredictor:
    def __init__(self, config):
        torch.cuda.is_available()
        self.torch_device = 'cuda' if torch.cuda.is_available() else 0 #'cpu'
        
        self.tokenizer = BartTokenizer.from_pretrained('bart-large-cnn')
        self.model = BartForConditionalGeneration.from_pretrained('bart-large-cnn')

    def predict(self, payload):
        article = payload["text"]
        article_input_ids = self.tokenizer.batch_encode_plus([article], return_tensors='pt', max_length=1024)['input_ids'].to(self.torch_device)
        summary_ids = self.model.cuda().generate(article_input_ids,
                             num_beams=4,
                             length_penalty=2.0,
                             max_length=60,
                             #min_len=40,
                             no_repeat_ngram_size=3)
        summary_txt = self.tokenizer.decode(summary_ids.squeeze(), skip_special_tokens=True)
        return summary_txt