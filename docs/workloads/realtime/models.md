# Models

## Model directory format

Whenever a model path is specified in an API configuration file, it should be a path to an S3/GCS prefix which contains your exported model. Directories may include a single model, or multiple folders each with a single model (note that a "single model" need not be a single file; there can be multiple files for a single model). When multiple folders are used, the folder names must be integer values, and will be interpreted as the model version. Model versions can be any integer, but are typically integer timestamps. It is always assumed that the highest version number is the latest version of your model.

Each predictor type expects a different model format:

### Python

For the Python predictor, any model structure is accepted. Here is an example:

```text
  s3://my-bucket/models/text-generator/
  ├── model.pkl
  └── data.txt
```

or for a versioned model:

```text
  s3://my-bucket/models/text-generator/
  ├── 1523423423/  (version number, usually a timestamp)
  |   ├── model.pkl
  |   └── data.txt
  └── 2434389194/  (version number, usually a timestamp)
      ├── model.pkl
      └── data.txt
```

### TensorFlow

For the TensorFlow predictor, the model path must be a SavedModel export:

```text
  s3://my-bucket/models/text-generator/
  ├── saved_model.pb
  └── variables/
      ├── variables.index
      ├── variables.data-00000-of-00003
      ├── variables.data-00001-of-00003
      └── variables.data-00002-of-...
```

or for a versioned model:

```text
  s3://my-bucket/models/text-generator/
  ├── 1523423423/  (version number, usually a timestamp)
  |   ├── saved_model.pb
  |   └── variables/
  |       ├── variables.index
  |       ├── variables.data-00000-of-00003
  |       ├── variables.data-00001-of-00003
  |       └── variables.data-00002-of-...
  └── 2434389194/  (version number, usually a timestamp)
      ├── saved_model.pb
      └── variables/
          ├── variables.index
          ├── variables.data-00000-of-00003
          ├── variables.data-00001-of-00003
          └── variables.data-00002-of-...
```

#### Inferentia

When Inferentia models are used, the directory structure is slightly different:

```text
  s3://my-bucket/models/text-generator/
  └── saved_model.pb
```

or for a versioned model:

```text
  s3://my-bucket/models/text-generator/
  ├── 1523423423/  (version number, usually a timestamp)
  |   └── saved_model.pb
  └── 2434389194/  (version number, usually a timestamp)
      └── saved_model.pb
```

### ONNX

For the ONNX predictor, the model path must contain a single `*.onnx` file:

```text
  s3://my-bucket/models/text-generator/
  └── model.onnx
```

or for a versioned model:

```text
  s3://my-bucket/models/text-generator/
  ├── 1523423423/  (version number, usually a timestamp)
  |   └── model.onnx
  └── 2434389194/  (version number, usually a timestamp)
      └── model.onnx
```

## Single model

The most common pattern is to serve a single model per API. The path to the model is specified in the `path` field in the `predictor.models` configuration. For example:

```yaml
# cortex.yaml

- name: iris-classifier
  kind: RealtimeAPI
  predictor:
    # ...
    models:
      path: s3://my-bucket/models/text-generator/
```

For the Python predictor type, the `models` field comes under the name of `multi_model_reloading`. It is also not necessary to specify the `multi_model_reloading` section at all, since you can download and load the model in your predictor's `__init__()` function. That said, it is necessary to use the `path` field to take advantage of [live model reloading](#live-model-reloading).

## Multiple models

It is possible to serve multiple models from a single API. The paths to the models are specified in the api configuration, either via the `models.paths` or `models.dir` field in the `predictor` configuration. For example:

```yaml
# cortex.yaml

- name: iris-classifier
  kind: RealtimeAPI
  predictor:
    # ...
    models:
      paths:
        - name: iris-classifier
          path: s3://my-bucket/models/text-generator/
        # ...
```

or:

```yaml
# cortex.yaml

- name: iris-classifier
  kind: RealtimeAPI
  predictor:
    # ...
    models:
      dir: s3://my-bucket/models/
```

For the Python predictor type, the `models` field comes under the name of `multi_model_reloading`. It is also not necessary to specify the `multi_model_reloading` section at all, since you can download and load the model in your predictor's `__init__()` function. That said, it is necessary to use the `models` field to take advantage of live model reloading or multi-model caching.

When using the `models.paths` field, each path must be a valid model directory (see above for valid model directory structures).

When using the `models.dir` field, the directory provided may contain multiple subdirectories, each of which is a valid model directory. For example:

```text
  s3://my-bucket/models/
  ├── text-generator
  |   └── * (model files)
  └── sentiment-analyzer
      ├── 24753823/
      |   └── * (model files)
      └── 26234288/
          └── * (model files)
```

In this case, there are two models in the directory, one of which is named "text-generator", and the other is named "sentiment-analyzer".

## Live model reloading

Live model reloading is a mechanism that periodically checks for updated models in the model path(s) provided in `predictor.models`. It is automatically enabled for all predictor types, including the Python predictor type (as long as model paths are specified via `multi_model_reloading` in the `predictor` configuration).

The following is a list of events that will trigger the API to update its model(s):

* A new model is added to the model directory.
* A model is removed from the model directory.
* A model changes its directory structure.
* A file in the model directory is updated in-place.

Usage varies based on the predictor type:

### Python

To use live model reloading with the Python predictor, the model path(s) must be specified in the API's `predictor` configuration, via the `models` field. When models are specified in this manner, your `PythonPredictor` class must implement the `load_model()` function, and models can be retrieved by using the `get_model()` method of the `python_client` that's passed into your predictor's constructor.

The `load_model()` function that you implement in your `PythonPredictor` can return anything that you need to make a prediction. There is one caveat: whatever the return value is, it must be unloadable from memory via the `del` keyword. The following frameworks have been tested to work:

* PyTorch (CPU & GPU)
* ONNX (CPU & GPU)
* Sklearn/MLFlow (CPU)
* Numpy (CPU)
* Pandas (CPU)
* Caffe (not tested, but should work on CPU & GPU)

Python data structures containing these types are also supported (e.g. lists and dicts).

The `load_model()` function takes a single argument, which is a path (on disk) to the model to be loaded. Your `load_model()` function is called behind the scenes by Cortex when you call the `python_client`'s `get_model()` method. Cortex is responsible for downloading your model from S3/GCS onto the local disk before calling `load_model()` with the local path. Whatever `load_model()` returns will be the exact return value of `python_client.get_model()`. Here is the schema for `python_client.get_model()`:

```python
def get_model(model_name, model_version):
    """
    Retrieve a model for inference.

    Args:
        model_name (optional): Name of the model to retrieve (when multiple models are deployed in an API).
            When predictor.models.paths is specified, model_name should be the name of one of the models listed in the API config.
            When predictor.models.dir is specified, model_name should be the name of a top-level directory in the models dir.
        model_version (string, optional): Version of the model to retrieve. Can be omitted or set to "latest" to select the highest version.

    Returns:
        The value that's returned by your predictor's load_model() method.
    """
```

Here's an example:

```python
class PythonPredictor:
    def __init__(self, config, python_client):
        self.client = python_client

    def load_model(self, model_path):
        # model_path is a path to your model's directory on disk
        return load_from_disk(model_path)

    def predict(self, payload):
      model = self.client.get_model()
      return model.predict(payload)
```

When multiple models are being served in an API, `python_client.get_model()` can accept a model name:

```python
class PythonPredictor:
    # ...

    def predict(self, payload, query_params):
      model = self.client.get_model(query_params["model"])
      return model.predict(payload)
```

`python_client.get_model()` can also accept a model version if a version other than the highest is desired:

```python
class PythonPredictor:
    # ...

    def predict(self, payload, query_params):
      model = self.client.get_model(query_params["model"], query_params["version"])
      return model.predict(payload)
```

### TensorFlow

When using the TensorFlow predictor, inference is performed by using the `predict()` method of the `tensorflow_client` that's passed to the predictor's constructor:

```python
def predict(model_input, model_name, model_version) -> dict:
    """
    Run prediction.

    Args:
        model_input: Input to the model.
        model_name (optional): Name of the model to retrieve (when multiple models are deployed in an API).
            When predictor.models.paths is specified, model_name should be the name of one of the models listed in the API config.
            When predictor.models.dir is specified, model_name should be the name of a top-level directory in the models dir.
        model_version (string, optional): Version of the model to retrieve. Can be omitted or set to "latest" to select the highest version.

    Returns:
        dict: TensorFlow Serving response converted to a dictionary.
    """
```

For example:

```python
class TensorFlowPredictor:
    def __init__(self, tensorflow_client, config):
        self.client = tensorflow_client

    def predict(self, payload):
      return self.client.predict(payload)
```

When multiple models are being served in an API, `tensorflow_client.predict()` can accept a model name:

```python
class TensorFlowPredictor:
    # ...

    def predict(self, payload, query_params):
      return self.client.predict(payload, query_params["model"])
```

`tensorflow_client.predict()` can also accept a model version if a version other than the highest is desired:

```python
class TensorFlowPredictor:
    # ...

    def predict(self, payload, query_params):
      return self.client.predict(payload, query_params["model"], query_params["version"])
```

Note: when using Inferentia models with the TensorFlow predictor, live model reloading is only supported if `predictor.processes_per_replica` is set to 1 (the default value).

### ONNX

When using the ONNX predictor, inference is performed by using the `predict()` method of the `onnx_client` that's passed to the predictor's constructor:

```python
def predict(model_input: Any, model_name: Optional[str] = None, model_version: str = "latest") -> Any:
    """
    Run prediction.

    Args:
        model_input: Input to the model.
        model_name (optional): Name of the model to retrieve (when multiple models are deployed in an API).
            When predictor.models.paths is specified, model_name should be the name of one of the models listed in the API config.
            When predictor.models.dir is specified, model_name should be the name of a top-level directory in the models dir.
        model_version (string, optional): Version of the model to retrieve. Can be omitted or set to "latest" to select the highest version.

    Returns:
        The prediction returned from the model.
    """
```

For example:

```python
class ONNXPredictor:
    def __init__(self, onnx_client, config):
        self.client = onnx_client

    def predict(self, payload):
      return self.client.predict(payload)
```

When multiple models are being served in an API, `onnx_client.predict()` can accept a model name:

```python
class ONNXPredictor:
    # ...

    def predict(self, payload, query_params):
      return self.client.predict(payload, query_params["model"])
```

`onnx_client.predict()` can also accept a model version if a version other than the highest is desired:

```python
class ONNXPredictor:
    # ...

    def predict(self, payload, query_params):
      return self.client.predict(payload, query_params["model"], query_params["version"])
```

You can also retrieve information about the model by calling the `onnx_client`'s `get_model()` method (it supports model name and model version arguments, like its `predict()` method). This can be useful for retrieving the model's input/output signatures. For example, `self.client.get_model()` might look like this:

```python
{
    "session": "<onnxruntime.InferenceSession model object>",
    "signatures": "<onnxruntime.InferenceSession model object>['session'].get_inputs()",
    "input_signatures": {
        "<signature-name>": {
            "shape": "<input shape>",
            "type": "<numpy type>"
        }
        ...
    }
}
```
