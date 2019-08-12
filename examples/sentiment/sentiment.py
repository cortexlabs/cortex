import numpy as np
import tensorflow as tf
import tensorflow_hub as hub
import bert
from bert import run_classifier
from bert import tokenization


def create_tokenizer_from_hub_module():
    """Get the vocab file and casing info from the Hub module."""
    with tf.Graph().as_default():
        bert_module = hub.Module("https://tfhub.dev/google/bert_uncased_L-12_H-768_A-12/1")
        tokenization_info = bert_module(signature="tokenization_info", as_dict=True)
        with tf.Session() as sess:
            vocab_file, do_lower_case = sess.run(
                [tokenization_info["vocab_file"], tokenization_info["do_lower_case"]]
            )

    return bert.tokenization.FullTokenizer(vocab_file=vocab_file, do_lower_case=do_lower_case)


tokenizer = create_tokenizer_from_hub_module()
labels = ["Negative", "Positive"]


def pre_inference(sample, metadata):
    input_feature = run_classifier.convert_single_example(
        0,  # example Idx
        run_classifier.InputExample(guid="", text_a=sample["input"], text_b=None, label=0),
        [0, 1],  # label IDs
        128,  # max len
        tokenizer,
    )
    return {"input_ids": input_feature.input_ids}


def post_inference(prediction, metadata):
    return {
        "sentiment": labels[prediction["response"]["labels"][0]],
        "probabilities": prediction["response"]["probabilities"],
    }
