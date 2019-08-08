import numpy as np
import json
from cortex.lib.log import get_logger
import tensorflow as tf
import tensorflow_hub as hub
import bert
from bert import tokenization

logger = get_logger()
labels = ["Negative", "Positive"]

BERT_MODEL_HUB = "https://tfhub.dev/google/bert_uncased_L-12_H-768_A-12/1"
max_seq_length = 128


def create_tokenizer_from_hub_module():
    """Get the vocab file and casing info from the Hub module."""
    with tf.Graph().as_default():
        bert_module = hub.Module(BERT_MODEL_HUB)
        tokenization_info = bert_module(signature="tokenization_info", as_dict=True)
        with tf.Session() as sess:
            vocab_file, do_lower_case = sess.run(
                [tokenization_info["vocab_file"], tokenization_info["do_lower_case"]]
            )

    return bert.tokenization.FullTokenizer(vocab_file=vocab_file, do_lower_case=do_lower_case)


tokenizer = create_tokenizer_from_hub_module()


def pre_inference(sample, metadata):
    tokens_a = tokenizer.tokenize(sample["input"][0])
    if len(tokens_a) > max_seq_length - 2:
        tokens_a = tokens_a[0 : (max_seq_length - 2)]
    tokens = []
    tokens.append("[CLS]")
    for token in tokens_a:
        tokens.append(token)
    tokens.append("[SEP]")
    input_ids = tokenizer.convert_tokens_to_ids(tokens)

    # Zero-pad up to the sequence length.
    while len(input_ids) < max_seq_length:
        input_ids.append(0)

    return {"input": input_ids}


def post_inference(prediction, metadata):
    return {
        "sentiment": labels[prediction["response"]["labels"][0]],
        "probabilities": prediction["response"]["probabilities"],
    }
