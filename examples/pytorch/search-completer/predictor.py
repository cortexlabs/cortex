import torch
import regex
import tqdm

roberta = torch.hub.load("pytorch/fairseq", "roberta.large")
roberta.eval()
roberta.cuda()


def predict(payload, metadata):
    predictions = roberta.fill_mask(payload["text"] + " <mask>", topk=5)
    return [prediction[0] for prediction in predictions]
