# Predictor implementation

The `AsyncAPI` kind currently only supports the `python` predictor type.

## Project files

Cortex makes all files in the project directory (i.e. the directory which contains `cortex.yaml`) available for use in
your Predictor implementation. Python bytecode files (`*.pyc`, `*.pyo`, `*.pyd`), files or folders that start with `.`,
and the api configuration file (e.g. `cortex.yaml`) are excluded.

The following files can also be added at the root of the project's directory:

* `.cortexignore` file, which follows the same syntax and behavior as
  a [.gitignore file](https://git-scm.com/docs/gitignore).
* `.env` file, which exports environment variables that can be used in the predictor. Each line of this file must follow
  the `VARIABLE=value` format.

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
    def __init__(self, config, metrics_client):
        """(Required) Called once before the API becomes available. Performs
        setup such as downloading/initializing the model or downloading a
        vocabulary.

        Args:
            config (required): Dictionary passed from API configuration (if
                specified). This may contain information on where to download
                the model and/or metadata.
            metrics_client (optional): The cortex metrics client, which allows
                you to push custom metrics in order to build custom dashboards
                in grafana.
        """
        pass

    def predict(self, payload, request_id):
        """(Required) Called once per request. Preprocesses the request payload
        (if necessary), runs inference, and postprocesses the inference output
        (if necessary).

        Args:
            payload (optional): The request payload (see below for the possible
                payload types).
            request_id (optional): The request id string that identifies a workload

        Returns:
            Prediction or a batch of predictions.
        """
        pass
```

For proper separation of concerns, it is recommended to use the constructor's `config` parameter for information such as
from where to download the model and initialization files, or any configurable model parameters. You define `config` in
your API configuration, and it is passed through to your Predictor's constructor.

Your API can accept requests with different types of payloads. Navigate to the [API requests](#api-requests) section to
learn about how headers can be used to change the type of `payload` that is passed into your `predict` method.

At this moment, the AsyncAPI `predict` method can only return `JSON`-parseable objects. Navigate to
the [API responses](#api-responses) section to learn about how to configure it.

## API requests

The type of the `payload` parameter in `predict(self, payload)` can vary based on the content type of the request.
The `payload` parameter is parsed according to the `Content-Type` header in the request. Here are the parsing rules (see
below for examples):

1. For `Content-Type: application/json`, `payload` will be the parsed JSON body.
1. For `Content-Type: text/plain`, `payload` will be a string. `utf-8` encoding is assumed, unless specified otherwise (
   e.g. via `Content-Type: text/plain; charset=us-ascii`)
1. For all other `Content-Type` values, `payload` will be the raw `bytes` of the request body.

Here are some examples:

### JSON data

#### Making the request

```bash
curl http://***.amazonaws.com/my-api \
    -X POST -H "Content-Type: application/json" \
    -d '{"key": "value"}'
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

```bash
curl http://***.amazonaws.com/my-api \
    -X POST -H "Content-Type: application/octet-stream" \
    --data-binary @object.pkl
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

    def predict(self, payload):
        img = Image.open(io.BytesIO(payload))  # read the payload bytes as an image
        print(img.size)
```

### Text data

#### Making the request

```bash
curl http://***.amazonaws.com/my-api \
    -X POST -H "Content-Type: text/plain" \
    -d "hello world"
```

#### Reading the payload

Since the `Content-Type: text/plain` header is used, the `payload` parameter will be a `string` object:

```python
class PythonPredictor:
    def __init__(self, config):
        pass

    def predict(self, payload):
        print(payload)  # prints "hello world"
```

## API responses

Currently, AsyncAPI responses of your `predict()` method have to be a JSON-serializable dictionary.

## Chaining APIs

It is possible to make requests from one API to another within a Cortex cluster. All running APIs are accessible from
within the predictor at `http://api-<api_name>:8888/predict`, where `<api_name>` is the name of the API you are making a
request to.

For example, if there is an api named `text-generator` running in the cluster, you could make a request to it from a
different API by using:

```python
import requests


class PythonPredictor:
    def predict(self, payload):
        response = requests.post("http://api-text-generator:8888/predict", json={"text": "machine learning is"})
        # ...
```

## Structured logging

You can use Cortex's logger in your predictor implemention to log in JSON. This will enrich your logs with Cortex's
metadata, and you can add custom metadata to the logs by adding key value pairs to the `extra` key when using the
logger. For example:

```python
...
from cortex_internal.lib.log import logger as log


class PythonPredictor:
    def predict(self, payload):
        log.info("received payload", extra={"payload": payload})
```

The dictionary passed in via the `extra` will be flattened by one level. e.g.

```text
{"asctime": "2021-01-19 15:14:05,291", "levelname": "INFO", "message": "received payload", "process": 235, "payload": "this movie is awesome"}
```

To avoid overriding essential Cortex metadata, please refrain from specifying the following extra keys: `asctime`
, `levelname`, `message`, `labels`, and `process`. Log lines greater than 5 MB in size will be ignored.
