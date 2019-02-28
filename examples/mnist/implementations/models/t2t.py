import tensorflow as tf
from tensor2tensor.utils import trainer_lib
from tensor2tensor import models  # pylint: disable=unused-import
from tensor2tensor import problems  # pylint: disable=unused-import
from tensor2tensor.data_generators import problem_hparams
from tensor2tensor.utils import registry


def create_estimator(run_config, model_config):
    hparams = trainer_lib.create_hparams("basic_fc_small")
    run_config.data_parallelism = None
    run_config.t2t_device_info = {"num_async_replicas": 1}
    problem = registry.problem("image_mnist")
    problem.eval_metrics = lambda: []
    p_hparams = problem.get_hparams(hparams)
    hparams.problem = problem
    hparams.problem_hparams = p_hparams
    hparams.warm_start_from = None

    estimator = trainer_lib.create_estimator("basic_fc_relu", hparams, run_config)
    return estimator


def transform_tensors(features, labels=None):
    features["inputs"] = tf.reshape(features["inputs"], [28, 28, 1])
    features["targets"] = tf.expand_dims(labels, -1)
    return features, labels
