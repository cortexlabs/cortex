from transformers import AutoModelWithLMHead, AutoTokenizer
import spacy
import subprocess
import json

class PythonPredictor:
    def __init__(self, config):
        subprocess.call("python -m spacy download en_core_web_sm".split(" "))
        import en_core_web_sm

        self.tokenizer = AutoTokenizer.from_pretrained("mrm8488/t5-base-finetuned-question-generation-ap")
        self.model = AutoModelWithLMHead.from_pretrained("mrm8488/t5-base-finetuned-question-generation-ap") 
        self.nlp = en_core_web_sm.load()

    def predict(self, payload):
        context = payload["context"]
        answer = payload["answer"]
        max_length = 64
        
        if "max_length" in payload:
            max_length = payload["max_length"]
        
        input_text = "answer: %s  context: %s </s>" % (answer, context)
        features = self.tokenizer([input_text], return_tensors='pt')
        
        output = self.model.generate(input_ids=features['input_ids'], 
                    attention_mask=features['attention_mask'],
                    max_length=max_length)
        
        result = self.tokenizer.decode(output[0])
        response = { "result": result }
        
        return json.dumps(response)
        