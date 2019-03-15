import tensorflow as tf
from tensor2tensor.utils import trainer_lib
from tensor2tensor import models  # pylint: disable=unused-import
from tensor2tensor import problems  # pylint: disable=unused-import
from tensor2tensor.data_generators import problem_hparams
from tensor2tensor.utils import registry
from tensor2tensor.data_generators import imdb


def create_estimator(run_config, model_config):
    # t2t expects these keys in run_config
    run_config.data_parallelism = None
    run_config.t2t_device_info = {"num_async_replicas": 1}

    hparams = trainer_lib.create_hparams("transformer_base_single_gpu")
    problem = registry.problem("sentiment_imdb")
    p_hparams = problem.get_hparams(hparams)
    hparams.problem = problem
    hparams.problem_hparams = p_hparams

    # don't need eval_metrics
    problem.eval_metrics = lambda: []

    # t2t expects this key
    hparams.warm_start_from = None

    # reduce memory load
    hparams.num_hidden_layers = 2
    hparams.hidden_size = 32
    hparams.filter_size = 32
    hparams.num_heads = 2
    hparams.batch_size = 64

    estimator = trainer_lib.create_estimator("transformer", hparams, run_config)
    return estimator


def transform_tensorflow(features, labels, model_config):
    hparams = model_config["hparams"]
    max_length = model_config["aggregates"]["max_review_length"]

    # t2t model performs flattening and expects this input key
    features["inputs"] = tf.reshape(features["embedding_input"], [max_length])

    # t2t expects this key and dimension
    features["targets"] = tf.expand_dims(labels, 0)

    return features, labels


@registry.register_problem
class SentimentIMDBCortex(imdb.SentimentIMDB):
    """IMDB sentiment classification, character level."""

    def feature_encoders(self, data_dir):
        print("yolo")
