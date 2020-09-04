# Predictor implementation

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

Once your model is [exported](../../guides/exporting.md), you can implement one of Cortex's Predictor classes to deploy your model. A Predictor is a Python class that describes how to initialize your model and use it to make predictions.

Which Predictor you use depends on how your model is exported:

* [TensorFlow Predictor](#tensorflow-predictor) if your model is exported as a TensorFlow `SavedModel`
* [ONNX Predictor](#onnx-predictor) if your model is exported in the ONNX format
* [Python Predictor](#python-predictor) for all other cases

## Project files

Cortex makes all files in the project directory (i.e. the directory which contains `cortex.yaml`) available for use in your Predictor implementation. Python bytecode files (`*.pyc`, `*.pyo`, `*.pyd`), files or folders that start with `.`, and the api configuration file (e.g. `cortex.yaml`) are excluded.

The following files can also be added at the root of the project's directory:

* `.cortexignore` file, which follows the same syntax and behavior as a [.gitignore file](https://git-scm.com/docs/gitignore).
* `.env` file, which exports environment variables that can be used in the predictor. Each line of this file must follow the `VARIABLE=value` format.

For example, if your directory looks like this:

```text
./my-classifier/
├── cortex.yaml
├── values.json
├── predictor.py
├── ...
└── requirements.txt
```

You can access `values.json` in your Predictor like this:

```python
import json

class PythonPredictor:
    def __init__(self, config):
        with open('values.json', 'r') as values_file:
            values = json.load(values_file)
        self.values = values
```

## Python Predictor

### Interface

```python
# initialization code and variables can be declared here in global scope

class PythonPredictor:
    def __init__(self, config, job_spec):
        """(Required) Called once during each worker initialization. Performs
        setup such as downloading/initializing the model or downloading a
        vocabulary.

        Args:
            config (required): Dictionary passed from API configuration (if
                specified) merged with configuration passed in with Job
                Submission API. If there are conflicting keys, values in
                configuration specified in Job submission takes precedence.
            job_spec (optional): Dictionary containing the submitted job
                request and additional information such as the job_id.
        """
        pass

    def predict(self, payload, batch_id):
        """(Required) Called once per batch. Preprocesses the batch payload (if
        necessary), runs inference, postprocesses the inference output (if
        necessary), and writes the predictions to storage (i.e. S3 or a
        database, if desired).

        Args:
            payload (required): a batch (i.e. a list of one or more samples).
            batch_id (optional): uuid assigned to this batch.
        Returns:
            Nothing
        """
        pass

    def on_job_complete(self):
        """(Optional) Called once after all batches in the job have been
        processed. Performs post job completion tasks such as aggregating
        results, executing web hooks, or triggering other jobs.
        """
        pass
```

For proper separation of concerns, it is recommended to use the constructor's `config` parameter for information such as from where to download the model and initialization files, or any configurable model parameters. You define `config` in your [API configuration](api-configuration.md), and it is passed through to your Predictor's constructor. The `config` parameters in the `API configuration` can be overridden by providing `config` in the job submission requests.

### Examples

<!-- CORTEX_VERSION_MINOR -->
You can find an example of a BatchAPI using a PythonPredictor in [examples/batch/image-classifier](https://github.com/cortexlabs/cortex/tree/master/examples/batch/image-classifier).

### Pre-installed packages

The following Python packages are pre-installed in Python Predictors and can be used in your implementations:

```text
boto3==1.13.7
cloudpickle==1.4.1
Cython==0.29.17
dill==0.3.1.1
fastapi==0.54.1
joblib==0.14.1
Keras==2.3.1
msgpack==1.0.0
nltk==3.5
np-utils==0.5.12.1
numpy==1.18.4
opencv-python==4.2.0.34
pandas==1.0.3
Pillow==7.1.2
pyyaml==5.3.1
requests==2.23.0
scikit-image==0.17.1
scikit-learn==0.22.2.post1
scipy==1.4.1
six==1.14.0
statsmodels==0.11.1
sympy==1.5.1
tensorflow-hub==0.8.0
tensorflow==2.1.0
torch==1.5.0
torchvision==0.6.0
xgboost==1.0.2
```

#### Inferentia-equipped APIs

The list is slightly different for inferentia-equipped APIs:

```text
boto3==1.13.7
cloudpickle==1.3.0
Cython==0.29.17
dill==0.3.1.1
fastapi==0.54.1
joblib==0.14.1
msgpack==1.0.0
neuron-cc==1.0.9410.0+6008239556
nltk==3.4.5
np-utils==0.5.12.1
numpy==1.16.5
opencv-python==4.2.0.32
pandas==1.0.3
Pillow==6.2.2
pyyaml==5.3.1
requests==2.23.0
scikit-image==0.16.2
scikit-learn==0.22.2.post1
scipy==1.3.2
six==1.14.0
statsmodels==0.11.1
sympy==1.5.1
tensorflow-neuron==1.15.0.1.0.1333.0
torch-neuron==1.0.825.0
torchvision==0.4.2
```

<!-- CORTEX_VERSION_MINOR x3 -->
The pre-installed system packages are listed in [images/python-predictor-cpu/Dockerfile](https://github.com/cortexlabs/cortex/tree/master/images/python-predictor-cpu/Dockerfile) (for CPU), [images/python-predictor-gpu/Dockerfile](https://github.com/cortexlabs/cortex/tree/master/images/python-predictor-gpu/Dockerfile) (for GPU), or [images/python-predictor-inf/Dockerfile](https://github.com/cortexlabs/cortex/tree/master/images/python-predictor-inf/Dockerfile) (for Inferentia).

If your application requires additional dependencies, you can install additional [Python packages](../python-packages.md) and [system packages](../system-packages.md).

## TensorFlow Predictor

### Interface

```python
class TensorFlowPredictor:
    def __init__(self, tensorflow_client, config, job_spec):
        """(Required) Called once during each worker initialization. Performs
        setup such as downloading/initializing the model or downloading a
        vocabulary.

        Args:
            tensorflow_client (required): TensorFlow client which is used to
                make predictions. This should be saved for use in predict().
            config (required): Dictionary passed from API configuration (if
                specified) merged with configuration passed in with Job
                Submission API. If there are conflicting keys, values in
                configuration specified in Job submission takes precedence.
            job_spec (optional): Dictionary containing the submitted job request
                and additional information such as the job_id.
        """
        self.client = tensorflow_client
        # Additional initialization may be done here

    def predict(self, payload, batch_id):
        """(Required) Called once per batch. Preprocesses the batch payload (if
        necessary), runs inference (e.g. by calling
        self.client.predict(model_input)), postprocesses the inference output
        (if necessary), and writes the predictions to storage (i.e. S3 or a
        database, if desired).

        Args:
            payload (required): a batch (i.e. a list of one or more samples).
            batch_id (optional): uuid assigned to this batch.
        Returns:
            Nothing
        """
        pass

    def on_job_complete(self):
        """(Optional) Called once after all batches in the job have been
        processed. Performs post job completion tasks such as aggregating
        results, executing web hooks, or triggering other jobs.
        """
        pass
```

<!-- CORTEX_VERSION_MINOR -->
Cortex provides a `tensorflow_client` to your Predictor's constructor. `tensorflow_client` is an instance of [TensorFlowClient](https://github.com/cortexlabs/cortex/tree/master/pkg/workloads/cortex/lib/client/tensorflow.py) that manages a connection to a TensorFlow Serving container to make predictions using your model. It should be saved as an instance variable in your Predictor, and your `predict()` function should call `tensorflow_client.predict()` to make an inference with your exported TensorFlow model. Preprocessing of the JSON payload and postprocessing of predictions can be implemented in your `predict()` function as well.

When multiple models are defined using the Predictor's `models` field, the `tensorflow_client.predict()` method expects a second argument `model_name` which must hold the name of the model that you want to use for inference (for example: `self.client.predict(payload, "text-generator")`). See the [multi model guide](../../guides/multi-model.md#tensorflow-predictor) for more information.

For proper separation of concerns, it is recommended to use the constructor's `config` parameter for information such as from where to download the model and initialization files, or any configurable model parameters. You define `config` in your [API configuration](api-configuration.md), and it is passed through to your Predictor's constructor. The `config` parameters in the `API configuration` can be overridden by providing `config` in the job submission requests.

### Examples

<!-- CORTEX_VERSION_MINOR -->
You can find an example of a BatchAPI using a TensorFlowPredictor in [examples/batch/tensorflow](https://github.com/cortexlabs/cortex/tree/master/examples/batch/tensorflow).

### Pre-installed packages

The following Python packages are pre-installed in TensorFlow Predictors and can be used in your implementations:

```text
boto3==1.13.7
dill==0.3.1.1
fastapi==0.54.1
msgpack==1.0.0
numpy==1.18.4
opencv-python==4.2.0.34
pyyaml==5.3.1
requests==2.23.0
tensorflow-hub==0.8.0
tensorflow-serving-api==2.1.0
tensorflow==2.1.0
```

<!-- CORTEX_VERSION_MINOR -->
The pre-installed system packages are listed in [images/tensorflow-predictor/Dockerfile](https://github.com/cortexlabs/cortex/tree/master/images/tensorflow-predictor/Dockerfile).

If your application requires additional dependencies, you can install additional [Python packages](../python-packages.md) and [system packages](../system-packages.md).

## ONNX Predictor

### Interface

```python
class ONNXPredictor:
    def __init__(self, onnx_client, config, job_spec):
        """(Required) Called once during each worker initialization. Performs
        setup such as downloading/initializing the model or downloading a
        vocabulary.

        Args:
            onnx_client (required): ONNX client which is used to make
                predictions. This should be saved for use in predict().
            config (required): Dictionary passed from API configuration (if
                specified) merged with configuration passed in with Job
                Submission API. If there are conflicting keys, values in
                configuration specified in Job submission takes precedence.
            job_spec (optional): Dictionary containing the submitted job request
                and additional information such as the job_id.
        """
        self.client = onnx_client
        # Additional initialization may be done here

    def predict(self, payload, batch_id):
        """(Required) Called once per batch. Preprocesses the batch payload (if
        necessary), runs inference (e.g. by calling
        self.client.predict(model_input)), postprocesses the inference output
        (if necessary), and writes the predictions to storage (i.e. S3 or a
        database, if desired).

        Args:
            payload (required): a batch (i.e. a list of one or more samples).
            batch_id (optional): uuid assigned to this batch.
        Returns:
            Nothing
        """
        pass

    def on_job_complete(self):
        """(Optional) Called once after all batches in the job have been
        processed. Performs post job completion tasks such as aggregating
        results, executing web hooks, or triggering other jobs.
        """
        pass
```

<!-- CORTEX_VERSION_MINOR -->
Cortex provides an `onnx_client` to your Predictor's constructor. `onnx_client` is an instance of [ONNXClient](https://github.com/cortexlabs/cortex/tree/master/pkg/workloads/cortex/lib/client/onnx.py) that manages an ONNX Runtime session to make predictions using your model. It should be saved as an instance variable in your Predictor, and your `predict()` function should call `onnx_client.predict()` to make an inference with your exported ONNX model. Preprocessing of the JSON payload and postprocessing of predictions can be implemented in your `predict()` function as well.

When multiple models are defined using the Predictor's `models` field, the `onnx_client.predict()` method expects a second argument `model_name` which must hold the name of the model that you want to use for inference (for example: `self.client.predict(model_input, "text-generator")`). See the [multi model guide](../../guides/multi-model.md#onnx-predictor) for more information.

For proper separation of concerns, it is recommended to use the constructor's `config` parameter for information such as from where to download the model and initialization files, or any configurable model parameters. You define `config` in your [API configuration](api-configuration.md), and it is passed through to your Predictor's constructor. The `config` parameters in the `API configuration` can be overridden by providing `config` in the job submission requests.

### Examples

<!-- CORTEX_VERSION_MINOR -->
You can find an example of a BatchAPI using an ONNXPredictor in [examples/batch/onnx](https://github.com/cortexlabs/cortex/tree/master/examples/batch/onnx).

### Pre-installed packages

The following Python packages are pre-installed in ONNX Predictors and can be used in your implementations:

```text
boto3==1.13.7
dill==0.3.1.1
fastapi==0.54.1
msgpack==1.0.0
numpy==1.18.4
onnxruntime==1.2.0
pyyaml==5.3.1
requests==2.23.0
```

<!-- CORTEX_VERSION_MINOR x2 -->
The pre-installed system packages are listed in [images/onnx-predictor-cpu/Dockerfile](https://github.com/cortexlabs/cortex/tree/master/images/onnx-predictor-cpu/Dockerfile) (for CPU) or [images/onnx-predictor-gpu/Dockerfile](https://github.com/cortexlabs/cortex/tree/master/images/onnx-predictor-gpu/Dockerfile) (for GPU).

If your application requires additional dependencies, you can install additional [Python packages](../python-packages.md) and [system packages](../system-packages.md).
