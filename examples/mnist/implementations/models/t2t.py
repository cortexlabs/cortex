import tensorflow as tf
from tensor2tensor.utils import trainer_lib
from tensor2tensor import models  # pylint: disable=unused-import
from tensor2tensor import problems  # pylint: disable=unused-import
from tensor2tensor.data_generators import problem_hparams
from tensor2tensor.utils import registry


def create_estimator(run_config, model_config):
    # t2t expects these keys in run_config
    run_config.data_parallelism = None
    run_config.t2t_device_info = {"num_async_replicas": 1}

    # t2t has its own set of hyperparameters we can use
    hparams = trainer_lib.create_hparams("basic_fc_small")
    problem = registry.problem("image_mnist")
    p_hparams = problem.get_hparams(hparams)
    hparams.problem = problem
    hparams.problem_hparams = p_hparams

    # don't need eval_metrics
    problem.eval_metrics = lambda: []

    # t2t expects this key
    hparams.warm_start_from = None

    estimator = trainer_lib.create_estimator("basic_fc_relu", hparams, run_config)
    return estimator


def transform_tensorflow(features, labels, model_config):
    hparams = model_config["hparams"]

    # t2t model performs flattening and expects this input key
    features["inputs"] = tf.reshape(features["image_pixels"], hparams["input_shape"])

    # t2t expects this key and dimensionality
    features["targets"] = tf.expand_dims(labels, 0)

    return features, labels
