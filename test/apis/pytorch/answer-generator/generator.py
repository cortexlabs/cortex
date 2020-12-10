# This file includes code which was modified from https://colab.research.google.com/drive/1KTLqiAOdKM_3RnBWfqgrvOQLqumUyOdA

import torch
import torch.nn.functional as F


END_OF_TEXT = 50256


def generate(model, conditioned_tokens, device):
    generated_tokens = []
    while True:
        result = recalc(model, conditioned_tokens, generated_tokens, device)
        if result == END_OF_TEXT:
            return generated_tokens[:-1]


def recalc(model, conditioned_tokens, generated_tokens, device):
    indexed_tokens = conditioned_tokens + generated_tokens
    tokens_tensor = torch.tensor([indexed_tokens])
    tokens_tensor = tokens_tensor.to(device)
    with torch.no_grad():
        outputs = model(tokens_tensor)
        predictions = outputs[0]
    logits = predictions[0, -1, :]
    filtered_logits = top_p_filtering(logits)
    probabilities = F.softmax(filtered_logits, dim=-1)
    next_token = torch.multinomial(probabilities, 1)
    generated_tokens.append(next_token.item())
    return next_token.item()


def top_p_filtering(logits, top_p=0.9, filter_value=-float("Inf")):
    assert logits.dim() == 1
    sorted_logits, sorted_indices = torch.sort(logits, descending=True)
    cumulative_probs = torch.cumsum(F.softmax(sorted_logits, dim=-1), dim=-1)
    sorted_indices_to_remove = cumulative_probs > top_p
    sorted_indices_to_remove[..., 1:] = sorted_indices_to_remove[..., :-1].clone()
    sorted_indices_to_remove[..., 0] = 0
    indices_to_remove = sorted_indices[sorted_indices_to_remove]
    logits[indices_to_remove] = filter_value
    return logits
