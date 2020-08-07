# Predictor implementation

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

Once your model is [exported](exporting.md), you can implement one of Cortex's Predictor classes to deploy your model. A Predictor is a Python class that describes how to initialize your model and use it to make predictions.

Which Predictor you use depends on how your model is exported:

* [TensorFlow Predictor](#tensorflow-predictor) if your model is exported as a TensorFlow `SavedModel`
* [ONNX Predictor](#onnx-predictor) if your model is exported in the ONNX format
* [Python Predictor](#python-predictor) for all other cases

The response type of the predictor can vary depending on your requirements, see [API responses](#api-responses) below.

## Project files

Cortex makes all files in the project directory (i.e. the directory which contains `cortex.yaml`) available for use in your Predictor implementation. Python bytecode files (`*.pyc`, `*.pyo`, `*.pyd`), files or folders that start with `.`, and the api configuration file (e.g. `cortex.yaml`) are excluded.

The following files can also be added at the root of the project's directory:

* `.cortexignore` file, which follows the same syntax and behavior as a [.gitignore file](https://git-scm.com/docs/gitignore).
* `.env` file, which exports environment variables that can be used in the predictor. Each line of this file must follow the `VARIABLE=value` format.

```text
./iris-classifier/
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
    def __init__(self, config, api_spec, job_spec):
        """Called once during each worker initialization. Performs setup such as downloading/initializing the model or downloading a vocabulary.

        Args:
            config: Dictionary passed from API configuration (if specified) merged with configuration passed in with Job Submission API. If there are conflicting keys, values in configuration specified in Job submission takes precedence.
            api_spec: Dictionary containing the yaml configuration specified in cortex.yaml (optional)
            job_spec: Dictionary containing the submitted job request and additional information such as the job_id (optional)
        """
        pass

    def predict(self, payload, batch_id):
        """Called once per batch. Preprocesses the batch payload (if necessary), runs inference, postprocesses the inference output (if necessary) and writes the predictions to storage (i.e. S3 or a database).

        Args:
            payload: a batch, a list of one or more samples (required).
            batch_id: uuid assigned to this batch
        Returns:
            Nothing if the function body writes the predictions to storage, otherwise, the prediction or batch of predictions
        """
        pass

    def on_job_complete(self):
        """Called once after all of the batches have been processed. Runs job completion processes such as coalescing results and triggers completion webhooks.
        """
        pass
```

For proper separation of concerns, it is recommended to use the constructor's `config` parameter for information such as from where to download the model and initialization files, or any configurable model parameters. You define `config` in your [API configuration](api-configuration.md), and it is passed through to your Predictor's constructor. The `config` parameters in the `API configuration` can be overriden by providing `config` in the job submission requests.

### Examples

<!-- CORTEX_VERSION_MINOR -->
Many of the [examples](https://github.com/cortexlabs/cortex/tree/master/examples) use the Python Predictor, including all of the PyTorch examples.

<!-- CORTEX_VERSION_MINOR -->
Here is the Predictor for [examples/pytorch/iris-classifier](https://github.com/cortexlabs/cortex/tree/master/examples/pytorch/iris-classifier):

```python
import re
import torch
import boto3
from model import IrisNet

labels = ["setosa", "versicolor", "virginica"]

class PythonPredictor: // TODO update this example
    def __init__(self, config):
        # download the model
        bucket, key = re.match("s3://(.+?)/(.+)", config["model"]).groups()
        s3 = boto3.client("s3")
        s3.download_file(bucket, key, "model.pth")

        # initialize the model
        model = IrisNet()
        model.load_state_dict(torch.load("model.pth"))
        model.eval()

        self.model = model

    def predict(self, payload):
        # Convert the request to a tensor and pass it into the model
        input_tensor = torch.FloatTensor(
            [
                [
                    payload["sepal_length"],
                    payload["sepal_width"],
                    payload["petal_length"],
                    payload["petal_width"],
                ]
            ]
        )

        # Run the prediction
        output = self.model(input_tensor)

        # Translate the model output to the corresponding label string
        return labels[torch.argmax(output[0])]
```

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

For Inferentia-equipped APIs, the list is slightly different:

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

If your application requires additional dependencies, you can install additional [Python packages](python-packages.md) and [system packages](system-packages.md).

## TensorFlow Predictor

### Interface

```python
class TensorFlowPredictor:
    def __init__(self, tensorflow_client, config, api_spec, job_spec):
        """Called once during each worker initialization. Performs setup such as downloading/initializing the model or downloading a vocabulary.

        Args:
            tensorflow_client: TensorFlow client which is used to make predictions. This should be saved for use in predict().
            config: Dictionary passed from API configuration (if specified) merged with configuration passed in with Job Submission API. If there are conflicting keys, values in configuration specified in Job submission takes precedence.
            api_spec: Dictionary containing the yaml configuration specified in cortex.yaml (optional)
            job_spec: Dictionary containing the submitted job request and additional information such as the job_id (optional)
        """
        self.client = tensorflow_client
        # Additional initialization may be done here

    def predict(self, payload, batch_id):
        """Called once per batch. Preprocesses the batch payload (if necessary), runs inference (e.g. by calling self.client.predict(model_input)), postprocesses the inference output (if necessary) and writes the predictions to storage (i.e. S3 or a database).

        Args:
            payload: a batch, a list of one or more samples (required).
            batch_id: uuid assigned to this batch
        Returns:
            Nothing if the function body writes the predictions to storage, otherwise, the prediction or batch of predictions
        """
        pass

    def on_job_complete(self):
        """Called once after all of the batches have been processed. Runs job completion processes such as coalescing results and triggers completion webhooks.
        """
        pass
```

<!-- CORTEX_VERSION_MINOR -->
Cortex provides a `tensorflow_client` to your Predictor's constructor. `tensorflow_client` is an instance of [TensorFlowClient](https://github.com/cortexlabs/cortex/tree/master/pkg/workloads/cortex/lib/client/tensorflow.py) that manages a connection to a TensorFlow Serving container to make predictions using your model. It should be saved as an instance variable in your Predictor, and your `predict()` function should call `tensorflow_client.predict()` to make an inference with your exported TensorFlow model. Preprocessing of the JSON payload and postprocessing of predictions can be implemented in your `predict()` function as well.

When multiple models are defined using the Predictor's `models` field, the `tensorflow_client.predict()` method expects a second argument `model_name` which must hold the name of the model that you want to use for inference (for example: `self.client.predict(payload, "iris-classifier")`). See the [multi model guide](../guides/multi-model.md#tensorflow-predictor) for more information.

For proper separation of concerns, it is recommended to use the constructor's `config` parameter for information such as from where to download the model and initialization files, or any configurable model parameters. You define `config` in your [API configuration](api-configuration.md), and it is passed through to your Predictor's constructor. The `config` parameters in the `API configuration` can be overriden by providing `config` in the job submission requests.

### Examples

<!-- CORTEX_VERSION_MINOR -->
Most of the examples in [examples/tensorflow](https://github.com/cortexlabs/cortex/tree/master/examples/tensorflow) use the TensorFlow Predictor.

<!-- CORTEX_VERSION_MINOR -->
Here is the Predictor for [examples/tensorflow/iris-classifier](https://github.com/cortexlabs/cortex/tree/master/examples/tensorflow/iris-classifier):

```python
labels = ["setosa", "versicolor", "virginica"]

class TensorFlowPredictor: // TODO update this example
    def __init__(self, tensorflow_client, config):
        self.client = tensorflow_client

    def predict(self, payload):
        prediction = self.client.predict(payload)
        predicted_class_id = int(prediction["class_ids"][0])
        return labels[predicted_class_id]
```

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

If your application requires additional dependencies, you can install additional [Python packages](python-packages.md) and [system packages](system-packages.md).

## ONNX Predictor

### Interface

```python
class ONNXPredictor:
    def __init__(self, onnx_client, config, api_spec, job_spec):
        """Called once during each worker initialization. Performs setup such as downloading/initializing the model or downloading a vocabulary.

        Args:
            onnx_client: ONNX client which is used to make predictions. This should be saved for use in predict().
            config: Dictionary passed from API configuration (if specified) merged with configuration passed in with Job Submission API. If there are conflicting keys, values in configuration specified in Job submission takes precedence.
            api_spec: Dictionary containing the yaml configuration specified in cortex.yaml (optional)
            job_spec: Dictionary containing the submitted job request and additional information such as the job_id (optional)
        """
        self.client = onnx_client
        # Additional initialization may be done here

    def predict(self, payload, batch_id):
        """Called once per batch. Preprocesses the batch payload (if necessary), runs inference (e.g. by calling self.client.predict(model_input)), postprocesses the inference output (if necessary) and writes the predictions to storage (i.e. S3 or a database).

        Args:
            payload: a batch, a list of one or more samples (required).
            batch_id: uuid assigned to this batch
        Returns:
            Nothing if the function body writes the predictions to storage, otherwise, the prediction or batch of predictions
        """
        pass

    def on_job_complete(self):
        """Called once after all of the batches have been processed. Runs job completion processes such as coalescing results and triggers completion webhooks.
        """
        pass
```

<!-- CORTEX_VERSION_MINOR -->
Cortex provides an `onnx_client` to your Predictor's constructor. `onnx_client` is an instance of [ONNXClient](https://github.com/cortexlabs/cortex/tree/master/pkg/workloads/cortex/lib/client/onnx.py) that manages an ONNX Runtime session to make predictions using your model. It should be saved as an instance variable in your Predictor, and your `predict()` function should call `onnx_client.predict()` to make an inference with your exported ONNX model. Preprocessing of the JSON payload and postprocessing of predictions can be implemented in your `predict()` function as well.

When multiple models are defined using the Predictor's `models` field, the `onnx_client.predict()` method expects a second argument `model_name` which must hold the name of the model that you want to use for inference (for example: `self.client.predict(model_input, "iris-classifier")`). See the [multi model guide](../guides/multi-model.md#onnx-predictor) for more information.

For proper separation of concerns, it is recommended to use the constructor's `config` parameter for information such as from where to download the model and initialization files, or any configurable model parameters. You define `config` in your [API configuration](api-configuration.md), and it is passed through to your Predictor's constructor. The `config` parameters in the `API configuration` can be overriden by providing `config` in the job submission requests.

### Examples

<!-- CORTEX_VERSION_MINOR -->
[examples/onnx/iris-classifier](https://github.com/cortexlabs/cortex/tree/master/examples/onnx/iris-classifier) uses the ONNX Predictor:

```python
labels = ["setosa", "versicolor", "virginica"]

class ONNXPredictor: // TODO update this example
    def __init__(self, onnx_client, config):
        self.client = onnx_client

    def predict(self, payload, batch_id):
        model_input = []
        for iris in payload:
            model_input.append([
                payload["sepal_length"],
                payload["sepal_width"],
                payload["petal_length"],
                payload["petal_width"],
            ])

        prediction = self.client.predict(model_input)
        predicted_class_id = prediction[0][0]
        return labels[predicted_class_id]
```

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

If your application requires additional dependencies, you can install additional [Python packages](python-packages.md) and [system packages](system-packages.md).
