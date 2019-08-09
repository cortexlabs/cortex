import numpy as np
import tensorflow as tf
import tensorflow_hub as hub
import bert
from bert import run_classifier
from bert import tokenization

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
    input_examples = [run_classifier.InputExample(guid="", text_a = x, text_b = None, label = 0) for x in sample["input"]]
    input_features = bert.run_classifier.convert_examples_to_features(input_examples, [0, 1], 128, tokenizer)
    return {
        'input_ids': input_features[0].input_ids,
    }


def post_inference(prediction, metadata):
    return {
        "sentiment": labels[prediction["response"]["labels"][0]],
        "probabilities": prediction["response"]["probabilities"],
    }
