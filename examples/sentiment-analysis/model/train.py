#!/usr/bin/env python
# coding: utf-8

# In[ ]:


# Modified source from https://colab.research.google.com/github/google-research/bert/blob/master/predicting_movie_reviews_with_bert_on_tf_hub.ipynb

# Copyright 2019 Google Inc.

# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at

#     http://www.apache.org/licenses/LICENSE-2.0

# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


# #Predicting Movie Review Sentiment with BERT on TF Hub

# If you’ve been following Natural Language Processing over the past year, you’ve probably heard of BERT: Bidirectional Encoder Representations from Transformers. It’s a neural network architecture designed by Google researchers that’s totally transformed what’s state-of-the-art for NLP tasks, like text classification, translation, summarization, and question answering.
#
# Now that BERT's been added to [TF Hub](https://www.tensorflow.org/hub) as a loadable module, it's easy(ish) to add into existing Tensorflow text pipelines. In an existing pipeline, BERT can replace text embedding layers like ELMO and GloVE. Alternatively, [finetuning](http://wiki.fast.ai/index.php/Fine_tuning) BERT can provide both an accuracy boost and faster training time in many cases.
#
# Here, we'll train a model to predict whether an IMDB movie review is positive or negative using BERT in Tensorflow with tf hub. Some code was adapted from [this colab notebook](https://colab.sandbox.google.com/github/tensorflow/tpu/blob/master/tools/colab/bert_finetuning_with_cloud_tpus.ipynb). Let's get started!

# In[ ]:


from sklearn.model_selection import train_test_split
import pandas as pd
import tensorflow as tf
import tensorflow_hub as hub
from datetime import datetime


# In addition to the standard libraries we imported above, we'll need to install BERT's python package.

# In[ ]:


get_ipython().system("pip install bert-tensorflow")


# In[ ]:


import bert
from bert import run_classifier
from bert import optimization
from bert import tokenization


# Below, we'll set an output directory location to store our model output and checkpoints. This can be a local directory, in which case you'd set OUTPUT_DIR to the name of the directory you'd like to create. If you're running this code in Google's hosted Colab, the directory won't persist after the Colab session ends.
#
# Alternatively, if you're a GCP user, you can store output in a GCP bucket. To do that, set a directory name in OUTPUT_DIR and the name of the GCP bucket in the BUCKET field.
#
# Set DO_DELETE to rewrite the OUTPUT_DIR if it exists. Otherwise, Tensorflow will load existing model checkpoints from that directory (if they exist).

# In[ ]:


# Set the output directory for saving model file
# Optionally, set a GCP bucket location

OUTPUT_DIR = "OUTPUT_DIR_NAME"  # @param {type:"string"}
# @markdown Whether or not to clear/delete the directory and create a new one
DO_DELETE = False  # @param {type:"boolean"}
# @markdown Set USE_BUCKET and BUCKET if you want to (optionally) store model output on GCP bucket.
USE_BUCKET = True  # @param {type:"boolean"}
BUCKET = "BUCKET_NAME"  # @param {type:"string"}

if USE_BUCKET:
    OUTPUT_DIR = "gs://{}/{}".format(BUCKET, OUTPUT_DIR)
    from google.colab import auth

    auth.authenticate_user()

if DO_DELETE:
    try:
        tf.gfile.DeleteRecursively(OUTPUT_DIR)
    except:
        # Doesn't matter if the directory didn't exist
        pass
tf.gfile.MakeDirs(OUTPUT_DIR)
print("***** Model output directory: {} *****".format(OUTPUT_DIR))


# #Data

# First, let's download the dataset, hosted by Stanford. The code below, which downloads, extracts, and imports the IMDB Large Movie Review Dataset, is borrowed from [this Tensorflow tutorial](https://www.tensorflow.org/hub/tutorials/text_classification_with_tf_hub).

# In[ ]:


from tensorflow import keras
import os
import re

# Load all files from a directory in a DataFrame.
def load_directory_data(directory):
    data = {}
    data["sentence"] = []
    data["sentiment"] = []
    for file_path in os.listdir(directory):
        with tf.gfile.GFile(os.path.join(directory, file_path), "r") as f:
            data["sentence"].append(f.read())
            data["sentiment"].append(re.match("\d+_(\d+)\.txt", file_path).group(1))
    return pd.DataFrame.from_dict(data)


# Merge positive and negative examples, add a polarity column and shuffle.
def load_dataset(directory):
    pos_df = load_directory_data(os.path.join(directory, "pos"))
    neg_df = load_directory_data(os.path.join(directory, "neg"))
    pos_df["polarity"] = 1
    neg_df["polarity"] = 0
    return pd.concat([pos_df, neg_df]).sample(frac=1).reset_index(drop=True)


# Download and process the dataset files.
def download_and_load_datasets(force_download=False):
    dataset = tf.keras.utils.get_file(
        fname="aclImdb.tar.gz",
        origin="http://ai.stanford.edu/~amaas/data/sentiment/aclImdb_v1.tar.gz",
        extract=True,
    )

    train_df = load_dataset(os.path.join(os.path.dirname(dataset), "aclImdb", "train"))
    test_df = load_dataset(os.path.join(os.path.dirname(dataset), "aclImdb", "test"))

    return train_df, test_df


# In[ ]:


train, test = download_and_load_datasets()


# To keep training fast, we'll take a sample of 5000 train and test examples, respectively.

# In[ ]:


train = train.sample(5000)
test = test.sample(5000)


# In[ ]:


train.columns


# For us, our input data is the 'sentence' column and our label is the 'polarity' column (0, 1 for negative and positive, respecitvely)

# In[ ]:


DATA_COLUMN = "sentence"
LABEL_COLUMN = "polarity"
# label_list is the list of labels, i.e. True, False or 0, 1 or 'dog', 'cat'
label_list = [0, 1]


# #Data Preprocessing
# We'll need to transform our data into a format BERT understands. This involves two steps. First, we create  `InputExample`'s using the constructor provided in the BERT library.
#
# - `text_a` is the text we want to classify, which in this case, is the `Request` field in our Dataframe.
# - `text_b` is used if we're training a model to understand the relationship between sentences (i.e. is `text_b` a translation of `text_a`? Is `text_b` an answer to the question asked by `text_a`?). This doesn't apply to our task, so we can leave `text_b` blank.
# - `label` is the label for our example, i.e. True, False

# In[ ]:


# Use the InputExample class from BERT's run_classifier code to create examples from the data
train_InputExamples = train.apply(
    lambda x: bert.run_classifier.InputExample(
        guid=None,  # Globally unique ID for bookkeeping, unused in this example
        text_a=x[DATA_COLUMN],
        text_b=None,
        label=x[LABEL_COLUMN],
    ),
    axis=1,
)

test_InputExamples = test.apply(
    lambda x: bert.run_classifier.InputExample(
        guid=None, text_a=x[DATA_COLUMN], text_b=None, label=x[LABEL_COLUMN]
    ),
    axis=1,
)


# Next, we need to preprocess our data so that it matches the data BERT was trained on. For this, we'll need to do a couple of things (but don't worry--this is also included in the Python library):
#
#
# 1. Lowercase our text (if we're using a BERT lowercase model)
# 2. Tokenize it (i.e. "sally says hi" -> ["sally", "says", "hi"])
# 3. Break words into WordPieces (i.e. "calling" -> ["call", "##ing"])
# 4. Map our words to indexes using a vocab file that BERT provides
# 5. Add special "CLS" and "SEP" tokens (see the [readme](https://github.com/google-research/bert))
# 6. Append "index" and "segment" tokens to each input (see the [BERT paper](https://arxiv.org/pdf/1810.04805.pdf))
#
# Happily, we don't have to worry about most of these details.
#
#
#

# To start, we'll need to load a vocabulary file and lowercasing information directly from the BERT tf hub module:

# In[ ]:


# This is a path to an uncased (all lowercase) version of BERT
BERT_MODEL_HUB = "https://tfhub.dev/google/bert_uncased_L-12_H-768_A-12/1"


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


# Great--we just learned that the BERT model we're using expects lowercase data (that's what stored in tokenization_info["do_lower_case"]) and we also loaded BERT's vocab file. We also created a tokenizer, which breaks words into word pieces:

# In[ ]:


tokenizer.tokenize("This here's an example of using the BERT tokenizer")


# Using our tokenizer, we'll call `run_classifier.convert_examples_to_features` on our InputExamples to convert them into features BERT understands.

# In[ ]:


# We'll set sequences to be at most 128 tokens long.
MAX_SEQ_LENGTH = 128
# Convert our train and test features to InputFeatures that BERT understands.
train_features = bert.run_classifier.convert_examples_to_features(
    train_InputExamples, label_list, MAX_SEQ_LENGTH, tokenizer
)
test_features = bert.run_classifier.convert_examples_to_features(
    test_InputExamples, label_list, MAX_SEQ_LENGTH, tokenizer
)


# #Creating a model
#
# Now that we've prepared our data, let's focus on building a model. `create_model` does just this below. First, it loads the BERT tf hub module again (this time to extract the computation graph). Next, it creates a single new layer that will be trained to adapt BERT to our sentiment task (i.e. classifying whether a movie review is positive or negative). This strategy of using a mostly trained model is called [fine-tuning](http://wiki.fast.ai/index.php/Fine_tuning).

# In[ ]:


def create_model(is_predicting, input_ids, input_mask, segment_ids, labels, num_labels):
    """Creates a classification model."""

    bert_module = hub.Module(BERT_MODEL_HUB, trainable=True)
    bert_inputs = dict(input_ids=input_ids, input_mask=input_mask, segment_ids=segment_ids)
    bert_outputs = bert_module(inputs=bert_inputs, signature="tokens", as_dict=True)

    # Use "pooled_output" for classification tasks on an entire sentence.
    # Use "sequence_outputs" for token-level output.
    output_layer = bert_outputs["pooled_output"]

    hidden_size = output_layer.shape[-1].value

    # Create our own layer to tune for politeness data.
    output_weights = tf.get_variable(
        "output_weights",
        [num_labels, hidden_size],
        initializer=tf.truncated_normal_initializer(stddev=0.02),
    )

    output_bias = tf.get_variable("output_bias", [num_labels], initializer=tf.zeros_initializer())

    with tf.variable_scope("loss"):

        # Dropout helps prevent overfitting
        output_layer = tf.nn.dropout(output_layer, keep_prob=0.9)

        logits = tf.matmul(output_layer, output_weights, transpose_b=True)
        logits = tf.nn.bias_add(logits, output_bias)
        log_probs = tf.nn.log_softmax(logits, axis=-1)

        # Convert labels into one-hot encoding
        one_hot_labels = tf.one_hot(labels, depth=num_labels, dtype=tf.float32)

        predicted_labels = tf.squeeze(tf.argmax(log_probs, axis=-1, output_type=tf.int32))
        # If we're predicting, we want predicted labels and the probabiltiies.
        if is_predicting:
            return (predicted_labels, log_probs)

        # If we're train/eval, compute loss between predicted and actual label
        per_example_loss = -tf.reduce_sum(one_hot_labels * log_probs, axis=-1)
        loss = tf.reduce_mean(per_example_loss)
        return (loss, predicted_labels, log_probs)


# Next we'll wrap our model function in a `model_fn_builder` function that adapts our model to work for training, evaluation, and prediction.

# In[ ]:


# model_fn_builder actually creates our model function
# using the passed parameters for num_labels, learning_rate, etc.
def model_fn_builder(num_labels, learning_rate, num_train_steps, num_warmup_steps):
    """Returns `model_fn` closure for TPUEstimator."""

    def model_fn(features, labels, mode, params):  # pylint: disable=unused-argument
        """The `model_fn` for TPUEstimator."""

        input_ids = features["input_ids"]
        input_mask = features["input_mask"]
        segment_ids = features["segment_ids"]
        label_ids = features["label_ids"]

        is_predicting = mode == tf.estimator.ModeKeys.PREDICT

        # TRAIN and EVAL
        if not is_predicting:

            (loss, predicted_labels, log_probs) = create_model(
                is_predicting, input_ids, input_mask, segment_ids, label_ids, num_labels
            )

            train_op = bert.optimization.create_optimizer(
                loss, learning_rate, num_train_steps, num_warmup_steps, use_tpu=False
            )

            # Calculate evaluation metrics.
            def metric_fn(label_ids, predicted_labels):
                accuracy = tf.metrics.accuracy(label_ids, predicted_labels)
                f1_score = tf.contrib.metrics.f1_score(label_ids, predicted_labels)
                auc = tf.metrics.auc(label_ids, predicted_labels)
                recall = tf.metrics.recall(label_ids, predicted_labels)
                precision = tf.metrics.precision(label_ids, predicted_labels)
                true_pos = tf.metrics.true_positives(label_ids, predicted_labels)
                true_neg = tf.metrics.true_negatives(label_ids, predicted_labels)
                false_pos = tf.metrics.false_positives(label_ids, predicted_labels)
                false_neg = tf.metrics.false_negatives(label_ids, predicted_labels)
                return {
                    "eval_accuracy": accuracy,
                    "f1_score": f1_score,
                    "auc": auc,
                    "precision": precision,
                    "recall": recall,
                    "true_positives": true_pos,
                    "true_negatives": true_neg,
                    "false_positives": false_pos,
                    "false_negatives": false_neg,
                }

            eval_metrics = metric_fn(label_ids, predicted_labels)

            if mode == tf.estimator.ModeKeys.TRAIN:
                return tf.estimator.EstimatorSpec(mode=mode, loss=loss, train_op=train_op)
            else:
                return tf.estimator.EstimatorSpec(
                    mode=mode, loss=loss, eval_metric_ops=eval_metrics
                )
        else:
            (predicted_labels, log_probs) = create_model(
                is_predicting, input_ids, input_mask, segment_ids, label_ids, num_labels
            )

            predictions = {"probabilities": log_probs, "labels": predicted_labels}
            return tf.estimator.EstimatorSpec(
                mode,
                export_outputs={"predict": tf.estimator.export.PredictOutput(predictions)},
                predictions=predictions,
            )

    # Return the actual model function in the closure
    return model_fn


# In[ ]:


# Compute train and warmup steps from batch size
# These hyperparameters are copied from this colab notebook (https://colab.sandbox.google.com/github/tensorflow/tpu/blob/master/tools/colab/bert_finetuning_with_cloud_tpus.ipynb)
BATCH_SIZE = 32
LEARNING_RATE = 2e-5
NUM_TRAIN_EPOCHS = 3.0
# Warmup is a period of time where hte learning rate
# is small and gradually increases--usually helps training.
WARMUP_PROPORTION = 0.1
# Model configs
SAVE_CHECKPOINTS_STEPS = 500
SAVE_SUMMARY_STEPS = 100


# In[ ]:


# Compute # train and warmup steps from batch size
num_train_steps = int(len(train_features) / BATCH_SIZE * NUM_TRAIN_EPOCHS)
num_warmup_steps = int(num_train_steps * WARMUP_PROPORTION)


# In[ ]:


# Specify outpit directory and number of checkpoint steps to save
run_config = tf.estimator.RunConfig(
    model_dir=OUTPUT_DIR,
    save_summary_steps=SAVE_SUMMARY_STEPS,
    save_checkpoints_steps=SAVE_CHECKPOINTS_STEPS,
)


# In[ ]:


model_fn = model_fn_builder(
    num_labels=len(label_list),
    learning_rate=LEARNING_RATE,
    num_train_steps=num_train_steps,
    num_warmup_steps=num_warmup_steps,
)

estimator = tf.estimator.Estimator(
    model_fn=model_fn, config=run_config, params={"batch_size": BATCH_SIZE}
)


# Next we create an input builder function that takes our training feature set (`train_features`) and produces a generator. This is a pretty standard design pattern for working with Tensorflow [Estimators](https://www.tensorflow.org/guide/estimators).

# In[ ]:


# Create an input function for training. drop_remainder = True for using TPUs.
train_input_fn = bert.run_classifier.input_fn_builder(
    features=train_features, seq_length=MAX_SEQ_LENGTH, is_training=True, drop_remainder=False
)


# Now we train our model! For me, using a Colab notebook running on Google's GPUs, my training time was about 14 minutes.

# In[ ]:


print(f"Beginning Training!")
current_time = datetime.now()
estimator.train(input_fn=train_input_fn, max_steps=num_train_steps)
print("Training took time ", datetime.now() - current_time)


# Now let's use our test data to see how well our model did:

# In[ ]:


test_input_fn = run_classifier.input_fn_builder(
    features=test_features, seq_length=MAX_SEQ_LENGTH, is_training=False, drop_remainder=False
)


# In[ ]:


estimator.evaluate(input_fn=test_input_fn, steps=None)


# Now let's write code to make predictions on new sentences:

# In[ ]:


def getPrediction(in_sentences):
    labels = ["Negative", "Positive"]
    input_examples = [
        run_classifier.InputExample(guid="", text_a=x, text_b=None, label=0) for x in in_sentences
    ]  # here, "" is just a dummy label
    input_features = run_classifier.convert_examples_to_features(
        input_examples, label_list, MAX_SEQ_LENGTH, tokenizer
    )
    predict_input_fn = run_classifier.input_fn_builder(
        features=input_features, seq_length=MAX_SEQ_LENGTH, is_training=False, drop_remainder=False
    )
    predictions = estimator.predict(predict_input_fn)
    return [
        (sentence, prediction["probabilities"], labels[prediction["labels"]])
        for sentence, prediction in zip(in_sentences, predictions)
    ]


# In[ ]:


pred_sentences = [
    "That movie was absolutely awful",
    "The acting was a bit lacking",
    "The film was creative and surprising",
    "Absolutely fantastic!",
]


# In[ ]:


predictions = getPrediction(pred_sentences)


# Voila! We have a sentiment classifier!

# In[ ]:


predictions


# In[ ]:


def serving_input_fn():
    reciever_tensors = {"input_ids": tf.placeholder(dtype=tf.int32, shape=[1, MAX_SEQ_LENGTH])}
    features = {
        "input_ids": reciever_tensors["input_ids"],
        "input_mask": 1 - tf.cast(tf.equal(reciever_tensors["input_ids"], 0), dtype=tf.int32),
        "segment_ids": tf.zeros(dtype=tf.int32, shape=[1, MAX_SEQ_LENGTH]),
        "label_ids": tf.placeholder(tf.int32, [None], name="label_ids"),
    }
    return tf.estimator.export.ServingInputReceiver(features, reciever_tensors)


estimator._export_to_tpu = False
estimator.export_savedmodel(OUTPUT_DIR + "/export", serving_input_fn)
