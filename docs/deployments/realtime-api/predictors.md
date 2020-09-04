# Predictor implementation

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

Once your model is [exported](../../guides/exporting.md), you can implement one of Cortex's Predictor classes to deploy your model. A Predictor is a Python class that describes how to initialize your model and use it to make predictions.

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
    def __init__(self, config):
        """(Required) Called once before the API becomes available. Performs
        setup such as downloading/initializing the model or downloading a
        vocabulary.

        Args:
            config (required): Dictionary passed from API configuration (if
                specified). This may contain information on where to download
                the model and/or metadata.
        """
        pass

    def predict(self, payload, query_params, headers):
        """(Required) Called once per request. Preprocesses the request payload
        (if necessary), runs inference, and postprocesses the inference output
        (if necessary).

        Args:
            payload (optional): The request payload (see below for the possible
                payload types).
            query_params (optional): A dictionary of the query parameters used
                in the request.
            headers (optional): A dictionary of the headers sent in the request.

        Returns:
            Prediction or a batch of predictions.
        """
        pass

    def post_predict(self, response, payload, query_params, headers):
        """(Optional) Called in the background after returning a response.
        Useful for tasks that the client doesn't need to wait on before
        receiving a response such as recording metrics or storing results.

        Args:
            response (optional): The response as returned by the predict method.
            payload (optional): The request payload (see below for the possible
                payload types).
            query_params (optional): A dictionary of the query parameters used
                in the request.
            headers (optional): A dictionary of the headers sent in the request.
        """
        pass
```

For proper separation of concerns, it is recommended to use the constructor's `config` parameter for information such as from where to download the model and initialization files, or any configurable model parameters. You define `config` in your [API configuration](api-configuration.md), and it is passed through to your Predictor's constructor.

Your API can accept requests with different types of payloads such as `JSON`-parseable, `bytes` or `starlette.datastructures.FormData` data. Navigate to the [API requests](#api-requests) section to learn about how headers can be used to change the type of `payload` that is passed into your `predict` method.

Your `predictor` method can return different types of objects such as `JSON`-parseable, `string`, and `bytes` objects. Navigate to the [API responses](#api-responses) section to learn about how to configure your `predictor` method to respond with different response codes and content-types.

### Examples

<!-- CORTEX_VERSION_MINOR -->
Many of the [examples](https://github.com/cortexlabs/cortex/tree/master/examples) use the Python Predictor, including all of the PyTorch examples.

<!-- CORTEX_VERSION_MINOR -->
Here is the Predictor for [examples/pytorch/text-generator](https://github.com/cortexlabs/cortex/tree/master/examples/pytorch/text-generator):

```python
import torch
from transformers import GPT2Tokenizer, GPT2LMHeadModel


class PythonPredictor:
    def __init__(self, config):
        self.device = "cuda" if torch.cuda.is_available() else "cpu"
        print(f"using device: {self.device}")
        self.tokenizer = GPT2Tokenizer.from_pretrained("gpt2")
        self.model = GPT2LMHeadModel.from_pretrained("gpt2").to(self.device)

    def predict(self, payload):
        input_length = len(payload["text"].split())
        tokens = self.tokenizer.encode(payload["text"], return_tensors="pt").to(self.device)
        prediction = self.model.generate(tokens, max_length=input_length + 20, do_sample=True)
        return self.tokenizer.decode(prediction[0])
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
    def __init__(self, tensorflow_client, config):
        """(Required) Called once before the API becomes available. Performs
        setup such as downloading/initializing a vocabulary.

        Args:
            tensorflow_client (required): TensorFlow client which is used to
                make predictions. This should be saved for use in predict().
            config (required): Dictionary passed from API configuration (if
                specified).
        """
        self.client = tensorflow_client
        # Additional initialization may be done here

    def predict(self, payload, query_params, headers):
        """(Required) Called once per request. Preprocesses the request payload
        (if necessary), runs inference (e.g. by calling
        self.client.predict(model_input)), and postprocesses the inference
        output (if necessary).

        Args:
            payload (optional): The request payload (see below for the possible
                payload types).
            query_params (optional): A dictionary of the query parameters used
                in the request.
            headers (optional): A dictionary of the headers sent in the request.

        Returns:
            Prediction or a batch of predictions.
        """
        pass

    def post_predict(self, response, payload, query_params, headers):
        """(Optional) Called in the background after returning a response.
        Useful for tasks that the client doesn't need to wait on before
        receiving a response such as recording metrics or storing results.

        Args:
            response (optional): The response as returned by the predict method.
            payload (optional): The request payload (see below for the possible
                payload types).
            query_params (optional): A dictionary of the query parameters used
                in the request.
            headers (optional): A dictionary of the headers sent in the request.
        """
        pass
```

<!-- CORTEX_VERSION_MINOR -->
Cortex provides a `tensorflow_client` to your Predictor's constructor. `tensorflow_client` is an instance of [TensorFlowClient](https://github.com/cortexlabs/cortex/tree/master/pkg/workloads/cortex/lib/client/tensorflow.py) that manages a connection to a TensorFlow Serving container to make predictions using your model. It should be saved as an instance variable in your Predictor, and your `predict()` function should call `tensorflow_client.predict()` to make an inference with your exported TensorFlow model. Preprocessing of the JSON payload and postprocessing of predictions can be implemented in your `predict()` function as well.

When multiple models are defined using the Predictor's `models` field, the `tensorflow_client.predict()` method expects a second argument `model_name` which must hold the name of the model that you want to use for inference (for example: `self.client.predict(payload, "text-generator")`). See the [multi model guide](../../guides/multi-model.md#tensorflow-predictor) for more information.

For proper separation of concerns, it is recommended to use the constructor's `config` parameter for information such as configurable model parameters or download links for initialization files. You define `config` in your [API configuration](api-configuration.md), and it is passed through to your Predictor's constructor.

Your API can accept requests with different types of payloads such as `JSON`-parseable, `bytes` or `starlette.datastructures.FormData` data. Navigate to the [API requests](#api-requests) section to learn about how headers can be used to change the type of `payload` that is passed into your `predict` method.

Your `predictor` method can return different types of objects such as `JSON`-parseable, `string`, and `bytes` objects. Navigate to the [API responses](#api-responses) section to learn about how to configure your `predictor` method to respond with different response codes and content-types.

### Examples

<!-- CORTEX_VERSION_MINOR -->
Most of the examples in [examples/tensorflow](https://github.com/cortexlabs/cortex/tree/master/examples/tensorflow) use the TensorFlow Predictor.

<!-- CORTEX_VERSION_MINOR -->
Here is the Predictor for [examples/tensorflow/iris-classifier](https://github.com/cortexlabs/cortex/tree/master/examples/tensorflow/iris-classifier):

```python
labels = ["setosa", "versicolor", "virginica"]

class TensorFlowPredictor:
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

If your application requires additional dependencies, you can install additional [Python packages](../python-packages.md) and [system packages](../system-packages.md).

## ONNX Predictor

### Interface

```python
class ONNXPredictor:
    def __init__(self, onnx_client, config):
        """(Required) Called once before the API becomes available. Performs
        setup such as downloading/initializing a vocabulary.

        Args:
            onnx_client (required): ONNX client which is used to make
                predictions. This should be saved for use in predict().
            config (required): Dictionary passed from API configuration (if
                specified).
        """
        self.client = onnx_client
        # Additional initialization may be done here

    def predict(self, payload, query_params, headers):
        """(Required) Called once per request. Preprocesses the request payload
        (if necessary), runs inference (e.g. by calling
        self.client.predict(model_input)), and postprocesses the inference
        output (if necessary).

        Args:
            payload (optional): The request payload (see below for the possible
                payload types).
            query_params (optional): A dictionary of the query parameters used
                in the request.
            headers (optional): A dictionary of the headers sent in the request.

        Returns:
            Prediction or a batch of predictions.
        """
        pass

    def post_predict(self, response, payload, query_params, headers):
        """(Optional) Called in the background after returning a response.
        Useful for tasks that the client doesn't need to wait on before
        receiving a response such as recording metrics or storing results.

        Args:
            response (optional): The response as returned by the predict method.
            payload (optional): The request payload (see below for the possible
                payload types).
            query_params (optional): A dictionary of the query parameters used
                in the request.
            headers (optional): A dictionary of the headers sent in the request.
        """
        pass
```

<!-- CORTEX_VERSION_MINOR -->
Cortex provides an `onnx_client` to your Predictor's constructor. `onnx_client` is an instance of [ONNXClient](https://github.com/cortexlabs/cortex/tree/master/pkg/workloads/cortex/lib/client/onnx.py) that manages an ONNX Runtime session to make predictions using your model. It should be saved as an instance variable in your Predictor, and your `predict()` function should call `onnx_client.predict()` to make an inference with your exported ONNX model. Preprocessing of the JSON payload and postprocessing of predictions can be implemented in your `predict()` function as well.

When multiple models are defined using the Predictor's `models` field, the `onnx_client.predict()` method expects a second argument `model_name` which must hold the name of the model that you want to use for inference (for example: `self.client.predict(model_input, "text-generator")`). See the [multi model guide](../../guides/multi-model.md#onnx-predictor) for more information.

For proper separation of concerns, it is recommended to use the constructor's `config` parameter for information such as configurable model parameters or download links for initialization files. You define `config` in your [API configuration](api-configuration.md), and it is passed through to your Predictor's constructor.

Your API can accept requests with different types of payloads such as `JSON`-parseable, `bytes` or `starlette.datastructures.FormData` data. Navigate to the [API requests](#api-requests) section to learn about how headers can be used to change the type of `payload` that is passed into your `predict` method.

Your `predictor` method can return different types of objects such as `JSON`-parseable, `string`, and `bytes` objects. Navigate to the [API responses](#api-responses) section to learn about how to configure your `predictor` method to respond with different response codes and content-types.

### Examples

<!-- CORTEX_VERSION_MINOR -->
[examples/onnx/iris-classifier](https://github.com/cortexlabs/cortex/tree/master/examples/onnx/iris-classifier) uses the ONNX Predictor:

```python
labels = ["setosa", "versicolor", "virginica"]

class ONNXPredictor:
    def __init__(self, onnx_client, config):
        self.client = onnx_client

    def predict(self, payload):
        model_input = [
            payload["sepal_length"],
            payload["sepal_width"],
            payload["petal_length"],
            payload["petal_width"],
        ]

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

If your application requires additional dependencies, you can install additional [Python packages](../python-packages.md) and [system packages](../system-packages.md).

## API requests

The type of the `payload` parameter in `predict(self, payload)` can vary based on the content type of the request. The `payload` parameter is parsed according to the `Content-Type` header in the request:

1. For `Content-Type: application/json`, `payload` will be the parsed JSON body.
1. For `Content-Type: multipart/form-data` / `Content-Type: application/x-www-form-urlencoded`, `payload` will be `starlette.datastructures.FormData` (key-value pairs where the value is a `string` for form data, or `starlette.datastructures.UploadFile` for file uploads, see [Starlette's documentation](https://www.starlette.io/requests/#request-files)).
1. For all other `Content-Type` values, `payload` will be the raw `bytes` of the request body.

The `payload` parameter type will be a Python object (*lists*, *dicts*, *numbers*) if a request with a JSON payload is made:

```bash
$ curl http://***.amazonaws.com/my-api \
    -X POST -H "Content-Type: application/json" \
    -d '{"key": "value"}'
```

The `payload` parameter type will be a `bytes` object if a request with a `Content-Type: application/octet-stream` is made:

```bash
$ curl http://***.amazonaws.com/my-api \
    -X POST -H "Content-Type: application/octet-stream" \
    -d @file.bin
```

The `payload` parameter type will be a `bytes` object if a request doesn't have the `Content-Type` set:

```bash
$ curl http://***.amazonaws.com/my-api \
    -X POST -H "Content-Type:" \
    -d @sample.txt
```

The `payload` parameter type will be a `starlette.datastructures.FormData` object if a request with a `Content-Type: multipart/form-data` is made:

```bash
$ curl http://***.amazonaws.com/my-api \
    -X POST -H "Content-Type: multipart/form-data" \
    -F "fieldName=@file.txt"
```

The `payload` parameter type will be a `starlette.datastructures.FormData` object if a request with a `Content-Type: application/x-www-form-urlencoded` is made:

```bash
$ curl http://***.amazonaws.com/my-api \
    -X POST -H "Content-Type: application/x-www-form-urlencoded" \
    -d @file.txt
```

The `payload` parameter type will be a `starlette.datastructures.FormData` object if no headers are added to the request:

```bash
$ curl http://***.amazonaws.com/my-api \
    -X POST \
    -d @file.txt
```

## API responses

The response of your `predict()` function may be:

1. A JSON-serializable object (*lists*, *dictionaries*, *numbers*, etc.)

2. A `string` object (e.g. `"class 1"`)

3. A `bytes` object (e.g. `bytes(4)` or `pickle.dumps(obj)`)

4. An instance of [starlette.responses.Response](https://www.starlette.io/responses/#response)

Here are some examples:

```python
def predict(self, payload):
    # json-serializable object
    response = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
    return response
```

```python
def predict(self, payload):
    # string object
    response = "class 1"
    return response
```

```python
def predict(self, payload):
    # bytes-like object
    array = np.random.randn(3, 3)
    response = pickle.dumps(array)
    return response
```

```python
def predict(self, payload):
    # starlette.responses.Response
    data = "class 1"
    response = starlette.responses.Response(
        content=data, media_type="text/plain")
    return response
```
