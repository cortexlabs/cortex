import spacy
import subprocess


class PythonPredictor:
    """
    Class to perform NER (named entity recognition)
    """

    def __init__(self, config):
        subprocess.call("python -m spacy download en_core_web_md".split(" "))
        import en_core_web_md

        self.nlp = en_core_web_md.load()

    def predict(self, payload):
        doc = self.nlp(payload["text"])
        proc = lambda ent: {"label": ent.label_, "start": ent.start, "end": ent.end}
        out = {ent.text: proc(ent) for ent in doc.ents}
        return out
