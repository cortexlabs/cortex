# Predictor implementation

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
    def __init__(self, config, python_client):
        """(Required) Called once before the API becomes available. Performs
        setup such as downloading/initializing the model or downloading a
        vocabulary.

        Args:
            config (required): Dictionary passed from API configuration (if
                specified). This may contain information on where to download
                the model and/or metadata.
            python_client (optional): Python client which is used to retrieve
                models for prediction. This should be saved for use in predict().
                Required when `predictor.model_path` or `predictor.models` is
                specified in the api configuration.
        """
        self.client = python_client # optional

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

        Note: post_predict() and predict() run in the same thread pool. The
        size of the thread pool can be increased by updating
        `threads_per_process` in the api configuration yaml.

        Args:
            response (optional): The response as returned by the predict method.
            payload (optional): The request payload (see below for the possible
                payload types).
            query_params (optional): A dictionary of the query parameters used
                in the request.
            headers (optional): A dictionary of the headers sent in the request.
        """
        pass

    def load_model(self, model_path):
        """(Optional) Called by Cortex to load a model when necessary.

        This method is required when `predictor.model_path` or `predictor.models`
        field is specified in the api configuration.

        Warning: this method must not make any modification to the model's
        contents on disk.

        Args:
            model_path: The path to the model on disk.

        Returns:
            The loaded model from disk. The returned object is what
            self.client.get_model() will return.
        """
        pass
```

<!-- CORTEX_VERSION_MINOR -->
When explicit model paths are specified in the Python predictor's API configuration, Cortex provides a `python_client` to your Predictor's constructor. `python_client` is an instance of [PythonClient](https://github.com/cortexlabs/cortex/tree/0.24/pkg/workloads/cortex/lib/client/python.py) that is used to load model(s) (it calls the `load_model()` method of your predictor, which must be defined when using explicit model paths). It should be saved as an instance variable in your Predictor, and your `predict()` function should call `python_client.get_model()` to load your model for inference. Preprocessing of the JSON payload and postprocessing of predictions can be implemented in your `predict()` function as well.

When multiple models are defined using the Predictor's `models` field, the `python_client.get_model()` method expects an argument `model_name` which must hold the name of the model that you want to load (for example: `self.client.get_model("text-generator")`). There is also an optional second argument to specify the model version. See [models](models.md) and the [multi model guide](../../guides/multi-model.md#python-predictor) for more information.

For proper separation of concerns, it is recommended to use the constructor's `config` parameter for information such as from where to download the model and initialization files, or any configurable model parameters. You define `config` in your API configuration, and it is passed through to your Predictor's constructor.

Your API can accept requests with different types of payloads such as `JSON`-parseable, `bytes` or `starlette.datastructures.FormData` data. Navigate to the [API requests](#api-requests) section to learn about how headers can be used to change the type of `payload` that is passed into your `predict` method.

Your `predictor` method can return different types of objects such as `JSON`-parseable, `string`, and `bytes` objects. Navigate to the [API responses](#api-responses) section to learn about how to configure your `predictor` method to respond with different response codes and content-types.

### Pre-installed packages

The following Python packages are pre-installed in Python Predictors and can be used in your implementations:

```text
boto3==1.14.53
cloudpickle==1.6.0
Cython==0.29.21
dill==0.3.2
fastapi==0.61.1
google-cloud-storage==1.32.0
joblib==0.16.0
Keras==2.4.3
msgpack==1.0.0
nltk==3.5
np-utils==0.5.12.1
numpy==1.19.1
opencv-python==4.4.0.42
pandas==1.1.1
Pillow==7.2.0
pyyaml==5.3.1
requests==2.24.0
scikit-image==0.17.2
scikit-learn==0.23.2
scipy==1.5.2
six==1.15.0
statsmodels==0.12.0
sympy==1.6.2
tensorflow-hub==0.9.0
tensorflow==2.3.0
torch==1.6.0
torchvision==0.7.0
xgboost==1.2.0
```

#### Inferentia-equipped APIs

The list is slightly different for inferentia-equipped APIs:

```text
boto3==1.13.7
cloudpickle==1.6.0
Cython==0.29.21
dill==0.3.1.1
fastapi==0.54.1
google-cloud-storage==1.32.0
joblib==0.16.0
msgpack==1.0.0
neuron-cc==1.0.20600.0+0.b426b885f
nltk==3.5
np-utils==0.5.12.1
numpy==1.18.2
opencv-python==4.4.0.42
pandas==1.1.1
Pillow==7.2.0
pyyaml==5.3.1
requests==2.23.0
scikit-image==0.17.2
scikit-learn==0.23.2
scipy==1.3.2
six==1.15.0
statsmodels==0.12.0
sympy==1.6.2
tensorflow==1.15.4
tensorflow-neuron==1.15.3.1.0.2043.0
torch==1.5.1
torch-neuron==1.5.1.1.0.1721.0
torchvision==0.6.1
```

<!-- CORTEX_VERSION_MINOR x3 -->
The pre-installed system packages are listed in [images/python-predictor-cpu/Dockerfile](https://github.com/cortexlabs/cortex/tree/0.24/images/python-predictor-cpu/Dockerfile) (for CPU), [images/python-predictor-gpu/Dockerfile](https://github.com/cortexlabs/cortex/tree/0.24/images/python-predictor-gpu/Dockerfile) (for GPU), or [images/python-predictor-inf/Dockerfile](https://github.com/cortexlabs/cortex/tree/0.24/images/python-predictor-inf/Dockerfile) (for Inferentia).

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

        Note: post_predict() and predict() run in the same thread pool. The
        size of the thread pool can be increased by updating
        `threads_per_process` in the api configuration yaml.

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
Cortex provides a `tensorflow_client` to your Predictor's constructor. `tensorflow_client` is an instance of [TensorFlowClient](https://github.com/cortexlabs/cortex/tree/0.24/pkg/workloads/cortex/lib/client/tensorflow.py) that manages a connection to a TensorFlow Serving container to make predictions using your model. It should be saved as an instance variable in your Predictor, and your `predict()` function should call `tensorflow_client.predict()` to make an inference with your exported TensorFlow model. Preprocessing of the JSON payload and postprocessing of predictions can be implemented in your `predict()` function as well.

When multiple models are defined using the Predictor's `models` field, the `tensorflow_client.predict()` method expects a second argument `model_name` which must hold the name of the model that you want to use for inference (for example: `self.client.predict(payload, "text-generator")`). There is also an optional third argument to specify the model version. See [models](models.md) and the [multi model guide](../../guides/multi-model.md#tensorflow-predictor) for more information.

For proper separation of concerns, it is recommended to use the constructor's `config` parameter for information such as configurable model parameters or download links for initialization files. You define `config` in your [API configuration](configuration.md), and it is passed through to your Predictor's constructor.

Your API can accept requests with different types of payloads such as `JSON`-parseable, `bytes` or `starlette.datastructures.FormData` data. Navigate to the [API requests](#api-requests) section to learn about how headers can be used to change the type of `payload` that is passed into your `predict` method.

Your `predictor` method can return different types of objects such as `JSON`-parseable, `string`, and `bytes` objects. Navigate to the [API responses](#api-responses) section to learn about how to configure your `predictor` method to respond with different response codes and content-types.

### Pre-installed packages

The following Python packages are pre-installed in TensorFlow Predictors and can be used in your implementations:

```text
boto3==1.14.53
dill==0.3.2
fastapi==0.61.1
google-cloud-storage==1.32.0
msgpack==1.0.0
numpy==1.19.1
opencv-python==4.4.0.42
pyyaml==5.3.1
requests==2.24.0
tensorflow-hub==0.9.0
tensorflow-serving-api==2.3.0
tensorflow==2.3.0
```

<!-- CORTEX_VERSION_MINOR -->
The pre-installed system packages are listed in [images/tensorflow-predictor/Dockerfile](https://github.com/cortexlabs/cortex/tree/0.24/images/tensorflow-predictor/Dockerfile).

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

        Note: post_predict() and predict() run in the same thread pool. The
        size of the thread pool can be increased by updating
        `threads_per_process` in the api configuration yaml.

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
Cortex provides an `onnx_client` to your Predictor's constructor. `onnx_client` is an instance of [ONNXClient](https://github.com/cortexlabs/cortex/tree/0.24/pkg/workloads/cortex/lib/client/onnx.py) that manages an ONNX Runtime session to make predictions using your model. It should be saved as an instance variable in your Predictor, and your `predict()` function should call `onnx_client.predict()` to make an inference with your exported ONNX model. Preprocessing of the JSON payload and postprocessing of predictions can be implemented in your `predict()` function as well.

When multiple models are defined using the Predictor's `models` field, the `onnx_client.predict()` method expects a second argument `model_name` which must hold the name of the model that you want to use for inference (for example: `self.client.predict(model_input, "text-generator")`). There is also an optional third argument to specify the model version. See [models](models.md) and the [multi model guide](../../guides/multi-model.md#onnx-predictor) for more information.

For proper separation of concerns, it is recommended to use the constructor's `config` parameter for information such as configurable model parameters or download links for initialization files. You define `config` in your [API configuration](configuration.md), and it is passed through to your Predictor's constructor.

Your API can accept requests with different types of payloads such as `JSON`-parseable, `bytes` or `starlette.datastructures.FormData` data. Navigate to the [API requests](#api-requests) section to learn about how headers can be used to change the type of `payload` that is passed into your `predict` method.

Your `predictor` method can return different types of objects such as `JSON`-parseable, `string`, and `bytes` objects. Navigate to the [API responses](#api-responses) section to learn about how to configure your `predictor` method to respond with different response codes and content-types.

### Pre-installed packages

The following Python packages are pre-installed in ONNX Predictors and can be used in your implementations:

```text
boto3==1.14.53
dill==0.3.2
fastapi==0.61.1
google-cloud-storage==1.32.0
msgpack==1.0.0
numpy==1.19.1
onnxruntime==1.4.0
pyyaml==5.3.1
requests==2.24.0
```

<!-- CORTEX_VERSION_MINOR x2 -->
The pre-installed system packages are listed in [images/onnx-predictor-cpu/Dockerfile](https://github.com/cortexlabs/cortex/tree/0.24/images/onnx-predictor-cpu/Dockerfile) (for CPU) or [images/onnx-predictor-gpu/Dockerfile](https://github.com/cortexlabs/cortex/tree/0.24/images/onnx-predictor-gpu/Dockerfile) (for GPU).

If your application requires additional dependencies, you can install additional [Python packages](../python-packages.md) and [system packages](../system-packages.md).

## API requests

The type of the `payload` parameter in `predict(self, payload)` can vary based on the content type of the request. The `payload` parameter is parsed according to the `Content-Type` header in the request. Here are the parsing rules (see below for examples):

1. For `Content-Type: application/json`, `payload` will be the parsed JSON body.
1. For `Content-Type: multipart/form-data` / `Content-Type: application/x-www-form-urlencoded`, `payload` will be `starlette.datastructures.FormData` (key-value pairs where the values are strings for text data, or `starlette.datastructures.UploadFile` for file uploads; see [Starlette's documentation](https://www.starlette.io/requests/#request-files)).
1. For all other `Content-Type` values, `payload` will be the raw `bytes` of the request body.

Here are some examples:

### JSON data

#### Making the request

##### Curl

```bash
$ curl https://***.amazonaws.com/my-api \
    -X POST -H "Content-Type: application/json" \
    -d '{"key": "value"}'
```

Or if you have a json file:

```bash
$ curl https://***.amazonaws.com/my-api \
    -X POST -H "Content-Type: application/json" \
    -d @file.json
```

##### Python

```python
import requests

url = "https://***.amazonaws.com/my-api"
requests.post(url, json={"key": "value"})
```

Or if you have a json string:

```python
import requests
import json

url = "https://***.amazonaws.com/my-api"
jsonStr = json.dumps({"key": "value"})
requests.post(url, data=jsonStr, headers={"Content-Type": "application/json"})
```

#### Reading the payload

When sending a JSON payload, the `payload` parameter will be a Python object:

```python
class PythonPredictor:
    def __init__(self, config):
        pass

    def predict(self, payload):
        print(payload["key"])  # prints "value"
```

### Binary data

#### Making the request

##### Curl

```bash
$ curl https://***.amazonaws.com/my-api \
    -X POST -H "Content-Type: application/octet-stream" \
    --data-binary @object.pkl
```

##### Python

```python
import requests
import pickle

url = "https://***.amazonaws.com/my-api"
pklBytes = pickle.dumps({"key": "value"})
requests.post(url, data=pklBytes, headers={"Content-Type": "application/octet-stream"})
```

#### Reading the payload

Since the `Content-Type: application/octet-stream` header is used, the `payload` parameter will be a `bytes` object:

```python
import pickle

class PythonPredictor:
    def __init__(self, config):
        pass

    def predict(self, payload):
        obj = pickle.loads(payload)
        print(obj["key"])  # prints "value"
```

Here's an example if the binary data is an image:

```python
from PIL import Image
import io

class PythonPredictor:
    def __init__(self, config):
        pass

    def predict(self, payload, headers):
        img = Image.open(io.BytesIO(payload))  # read the payload bytes as an image
        print(img.size)
```

### Form data (files)

#### Making the request

##### Curl

```bash
$ curl https://***.amazonaws.com/my-api \
    -X POST \
    -F "text=@text.txt" \
    -F "object=@object.pkl" \
    -F "image=@image.png"
```

##### Python

```python
import requests
import pickle

url = "https://***.amazonaws.com/my-api"
files = {
    "text": open("text.txt", "rb"),
    "object": open("object.pkl", "rb"),
    "image": open("image.png", "rb"),
}

requests.post(url, files=files)
```

#### Reading the payload

When sending files via form data, the `payload` parameter will be `starlette.datastructures.FormData` (key-value pairs where the values are `starlette.datastructures.UploadFile`, see [Starlette's documentation](https://www.starlette.io/requests/#request-files)). Either `Content-Type: multipart/form-data` or `Content-Type: application/x-www-form-urlencoded` can be used (typically `Content-Type: multipart/form-data` is used for files, and is the default in the examples above).

```python
from PIL import Image
import pickle

class PythonPredictor:
    def __init__(self, config):
        pass

    def predict(self, payload):
        text = payload["text"].file.read()
        print(text.decode("utf-8"))  # prints the contents of text.txt

        obj = pickle.load(payload["object"].file)
        print(obj["key"])  # prints "value" assuming `object.pkl` is a pickled dictionary {"key": "value"}

        img = Image.open(payload["image"].file)
        print(img.size)  # prints the dimensions of image.png
```

### Form data (text)

#### Making the request

##### Curl

```bash
$ curl https://***.amazonaws.com/my-api \
    -X POST \
    -d "key=value"
```

##### Python

```python
import requests

url = "https://***.amazonaws.com/my-api"
requests.post(url, data={"key": "value"})
```

#### Reading the payload

When sending text via form data, the `payload` parameter will be `starlette.datastructures.FormData` (key-value pairs where the values are strings, see [Starlette's documentation](https://www.starlette.io/requests/#request-files)). Either `Content-Type: multipart/form-data` or `Content-Type: application/x-www-form-urlencoded` can be used (typically `Content-Type: application/x-www-form-urlencoded` is used for text, and is the default in the examples above).

```python
class PythonPredictor:
    def __init__(self, config):
        pass

    def predict(self, payload):
        print(payload["key"])  # will print "value"
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

## Chaining APIs

It is possible to make requests from one API to another within a Cortex cluster. All running APIs are accessible from within the predictor at `http://api-<api_name>:8888/predict`, where `<api_name>` is the name of the API you are making a request to.

For example, if there is an api named `text-generator` running in the cluster, you could make a request to it from a different API by using:

```python
import requests

class PythonPredictor:
    def predict(self, payload):
        response = requests.post("http://api-text-generator:8888/predict", json={"text": "machine learning is"})
        # ...
```

Note that the autoscaling configuration (i.e. `target_replica_concurrency`) for the API that is making the request should be modified with the understanding that requests will still be considered "in-flight" with the first API as the request is being fulfilled in the second API (during which it will also be considered "in-flight" with the second API). See more details in the [autoscaling docs](autoscaling.md).
