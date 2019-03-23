import tensorflow as tf
from tensor2tensor.utils import trainer_lib
from tensor2tensor import models  # pylint: disable=unused-import
from tensor2tensor import problems  # pylint: disable=unused-import
from tensor2tensor.data_generators import problem_hparams
from tensor2tensor.utils import registry
from tensor2tensor.utils import metrics
from tensor2tensor.data_generators import imdb
from tensor2tensor.data_generators import text_encoder


def create_estimator(run_config, model_config):
    # t2t expects these keys in run_config
    run_config.data_parallelism = None
    run_config.t2t_device_info = {"num_async_replicas": 1}

    hparams = trainer_lib.create_hparams("transformer_base_single_gpu")

    problem = SentimentIMDBCortex()
    vocab = model_config["aggregates"]["reviews_vocab"]
    problem.set_vocab(list(vocab.keys()))
    p_hparams = problem.get_hparams(hparams)
    hparams.problem = problem
    hparams.problem_hparams = p_hparams

    # don't need eval_metrics
    problem.eval_metrics = lambda: [metrics.Metrics.ACC]

    # t2t expects this key
    hparams.warm_start_from = None

    # reduce memory load
    hparams.num_hidden_layers = 2
    hparams.hidden_size = 32
    hparams.filter_size = 32
    hparams.num_heads = 2

    estimator = trainer_lib.create_estimator("transformer", hparams, run_config)
    return estimator


def transform_tensorflow(features, labels, model_config):
    max_length = model_config["aggregates"]["max_review_length"]

    features["inputs"] = tf.expand_dims(
        tf.expand_dims(tf.reshape(features["embedding_input"], [max_length]), -1), -1
    )

    features["targets"] = tf.expand_dims(tf.expand_dims(labels, -1), -1)

    return features, labels


class SentimentIMDBCortex(imdb.SentimentIMDB):
    """IMDB sentiment classification, character level."""

    def set_vocab(self, vocab_list):
        self.vocab = vocab_list

    def feature_encoders(self, data_dir):
        encoder = text_encoder.TokenTextEncoder(vocab_filename=None, vocab_list=self.vocab)

        return {
            "inputs": encoder,
            "targets": text_encoder.ClassLabelEncoder(self.class_labels(data_dir)),
        }
