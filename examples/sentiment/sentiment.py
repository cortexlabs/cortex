import tensorflow as tf
import tensorflow_hub as hub
from bert import tokenization, run_classifier

with tf.Graph().as_default():
    bert_module = hub.Module("https://tfhub.dev/google/bert_uncased_L-12_H-768_A-12/1")
    tokenization_info = bert_module(signature="tokenization_info", as_dict=True)
    vocab_file = tokenization_info["vocab_file"]
    do_lower_case = tokenization_info["do_lower_case"]
    with tf.Session() as sess:
        vocab_file, do_lower_case = sess.run([vocab_file, do_lower_case])

tokenizer = tokenization.FullTokenizer(vocab_file=vocab_file, do_lower_case=do_lower_case)


def pre_inference(sample, metadata):
    input_example = run_classifier.InputExample(guid="", text_a=sample["input"], label=0)
    input_feature = run_classifier.convert_single_example(0, input_example, [0, 1], 128, tokenizer)
    return {"input_ids": [input_feature.input_ids]}


def post_inference(prediction, metadata):
    labels = ["negative", "positive"]
    return {"sentiment": labels[prediction["response"]["labels"][0]]}
