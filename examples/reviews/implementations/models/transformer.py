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
    hparams = trainer_lib.create_hparams("transformer_base_single_gpu")

    problem = SentimentIMDBCortex(list(model_config["aggregates"]["reviews_vocab"]))
    hparams.problem = problem
    hparams.problem_hparams = problem.get_hparams(hparams)

    problem.eval_metrics = lambda: [
        metrics.Metrics.ACC_TOP5,
        metrics.Metrics.ACC_PER_SEQ,
        metrics.Metrics.NEG_LOG_PERPLEXITY,
    ]

    # reduce memory load
    hparams.num_hidden_layers = 2
    hparams.hidden_size = 32
    hparams.filter_size = 32
    hparams.num_heads = 2

    # t2t expects these keys
    hparams.warm_start_from = None
    run_config.data_parallelism = None
    run_config.t2t_device_info = {"num_async_replicas": 1}

    estimator = trainer_lib.create_estimator("transformer", hparams, run_config)
    return estimator


def transform_tensorflow(features, labels, model_config):
    max_length = model_config["aggregates"]["max_review_length"]

    features["inputs"] = tf.expand_dims(tf.reshape(features["embedding_input"], [max_length]), -1)
    features["targets"] = tf.expand_dims(tf.expand_dims(labels, -1), -1)

    return features, labels


class SentimentIMDBCortex(imdb.SentimentIMDB):
    """IMDB sentiment classification, with an in-memory vocab"""

    def __init__(self, vocab_list):
        super().__init__()
        self.vocab = vocab_list

    def feature_encoders(self, data_dir):
        encoder = text_encoder.TokenTextEncoder(vocab_filename=None, vocab_list=self.vocab)

        return {
            "inputs": encoder,
            "targets": text_encoder.ClassLabelEncoder(self.class_labels(data_dir)),
        }
