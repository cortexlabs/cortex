# Models

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

The Cortex API config supports specifying models for all types of predictors.

Models loaded using the `predictor.model_path` or `predictor.models` fields benefit from having the live reloading functionality enabled. Specifying the `cache_size` and `disk_cache_size` fields when using the `predictor.models` field enables the model caching mechanism.

The following is a table showing when live reloading / model caching is possible.

| Predictor Type | Processes / API replica   | Live Reloading                   | Model Caching |
|----------------|---------------------------|----------------------------------|---------------|
| Python         | processes_per_replica = 1 | yes                              | yes           |
|                | processes_per_replica > 1 | yes                              | no            |
| TensorFlow     | processes_per_replica = 1 | yes                              | yes           |
|                | processes_per_replica > 1 | yes (no when inferentia is used) | no            |
| ONNX           | processes_per_replica = 1 | yes                              | yes           |
|                | processes_per_replica > 1 | yes                              | no            |

## Live model reloading

Live reloading is a mechanism present on all predictor types that periodically checks the S3 model paths provided through `predictor.model_path` or `predictor.models` fields for updated models. This happens on each API replica and it's totally transparent to the user.

The following is a list of events that trigger the API replica to update its models:
* When a new model is added to the S3 upstream.
* When a model is removed from the S3 upstream.
* When a model changes its directory structure on the S3 upstream.
* When a model gets an updated file on the S3 upstream.

### Usage

#### Python

For the Python predictor, specifying a model is optional to the user. But if it's decided to have a model specified using the API spec config, then retrieving the model for inference is done with the `python_client` client that's passed to the predictor's constructor. This also implies the implementation of a `load_model` method for the `PythonPredictor` class.

To loader model that needs to be implemented has this prototype.

```python
class PythonPredictor:
    def __init__(self, config, python_client):
        self.client = python_client
        # ...

    def load_model(self, model_path: str):
        # load model from given model_path disk path
        return model

    # ...
```

The `load_model` method has been tested to work with the following frameworks:
* ONNX (CPU + GPU)
* PyTorch (CPU + GPU)
* Sklearn/MLFlow (CPU)
* Numpy (CPU)
* Pandas (CPU)
* Caffe (not tested, but should work on the CPU + GPU)

To retrieve a model, use the `get_model` method of `python_client`.

```python
def get_model(model_name: Optional[str] = None, model_version: str = "latest") -> Any:
    """
    Retrieve model for inference.

    Args:
        model_name: Model to use when multiple models are deployed in a single API.
        model_version: Model version to use. Can also be "latest" for picking the highest version.

    Returns:
        The model as loaded by load_model method.
    """
```

When the `predictor.model_path` field is used, you can retrieve the model by running `python_client.get_model()`, because there's only one `model_name` in and because `model_version` is already set to `latest`, which means it will be picking up the model with the highest version number (if applicable).

When the `predictor.models.paths` field is used, the `model_name` is the name of the model that it has been given in the API spec config.

#### TensorFlow

For the TensorFlow predictor, to run a prediction, you need to call the `tensorflow_client.predict` method of the `tensorflow_client` client that's passed to the predictor's constructor.

```python
def predict(model_input: Any, model_name: Optional[str] = None, model_version: str = "latest") -> dict:
    """
    Run prediction.

    Args:
        model_input: Input to the model.
        model_name: Model to use when multiple models are deployed in a single API.
        model_version: Model version to use. Can also be "latest" for picking the highest version.

    Returns:
        dict: Prediction.
    """
```

When the `predictor.model_path` field is used, you can retrieve the model by running `tensorflow_client.predict()`, because there's only one `model_name` in and because `model_version` is already set to `latest`, which means it will be picking up the model with the highest version number (if applicable).

When the `predictor.models.paths` field is used, the `model_name` is the name of the model that is has been given in the API spec config.

#### ONNX

For the ONNX predictor, to run a prediction, you need to call the `onnx_client.predict` method of the `onnx_client` client that's passed to the predictor's constructor.

```python
def predict(model_input: Any, model_name: Optional[str] = None, model_version: str = "latest") -> Any:
    """
    Run prediction.

    Args:
        model_input: Input to the model.
        model_name: Model to use when multiple models are deployed in a single API.
        model_version: Model version to use. Can also be "latest" for picking the highest version.

    Returns:
        The prediction returned from the model.
    """
```

When the `predictor.model_path` field is used, you can retrieve the model by running `onnx_client.predict()`, because there's only one `model_name` in and because `model_version` is already set to `latest`, which means it will be picking up the model with the highest version number (if applicable).

When the `predictor.models.paths` field is used, the `model_name` is the name of the model that it has been given in the API spec config.

You can also retrieve the model by calling the `onnx_client.get_model` method - same arguments as for the `predict` method. This can be useful for retrieving the model's input/output signatures. Here's how the retrieved metadata can look like:

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

### Individual models

When specifying a model like in

```yaml
- name: example-api-name
  kind: RealtimeAPI
  predictor:
    ...
    model_path: s3://my-bucket/models/model-A/
```

or in

```yaml
- name: example-api-name
  kind: RealtimeAPI
  predictor:
    ...
    models:
      paths:
        - name: model-A
          model_path: s3://my-bucket/models/model-A/
```

the directory structure that each model must take depends on the predictor type that's in use.

#### Python

For the Python predictor, any structure to the model (or to the versioned model) is accepted.

Example of accepted directory structure:

```text
  models/model-A/
  ├── MLmodel
  ├── conda.yaml
  └── model.pkl
```

or for a versioned model:

```text
  models/model-A/
  ├── 1523423423/ (Version prefix, a timestamp)
  |   ├── MLmodel
  |   ├── conda.yaml
  |   └── model.pkl
  └── 2434389194/ (Version prefix, a timestamp)
      ├── MLmodel
      ├── conda.yaml
      └── model.pkl
```

#### TensorFlow

For the TensorFlow predictor, every model (or each versioned model) must be a SavedModel export.

Example of accepted directory structure:

```text
  models/model-A/
  ├── saved_model.pb
  └── variables/
      ├── variables.index
      ├── variables.data-00000-of-00003
      ├── variables.data-00001-of-00003
      └── variables.data-00002-of-...
```

or for a versioned model:

```text
  models/model-A/
  ├── 1523423423/ (Version prefix, a timestamp)
  |   ├── saved_model.pb
  |   └── variables/
  |       ├── variables.index
  |       ├── variables.data-00000-of-00003
  |       ├── variables.data-00001-of-00003
  |       └── variables.data-00002-of-...
  └── 2434389194/ (Version prefix, a timestamp)
      ├── saved_model.pb
      └── variables/
          ├── variables.index
          ├── variables.data-00000-of-00003
          ├── variables.data-00001-of-00003
          └── variables.data-00002-of-...
```

##### Note for Inferentia

When Inferentia-exported models are used, the accepted directory structure is slightly different.

Accepted directory structure:

```text
  models/model-A/
  └── saved_model.pb
```

or for a versioned model:

```text
  models/model-A/
  ├── 1523423423/ (Version prefix, a timestamp)
  |   └── saved_model.pb
  └── 2434389194/ (Version prefix, a timestamp)
      └── saved_model.pb
```

#### ONNX

For the ONNX predictor, every model (or each versioned model) must contain a single `*.onnx` file.

Example of accepted directory structure:

```text
  models/model-A/
  └── model.onnx
```

or for a versioned model:

```text
  models/model-A/
  ├── 1523423423/ (Version prefix, a timestamp)
  |   └── model.onnx
  └── 2434389194/ (Version prefix, a timestamp)
      └── model.onnx
```

### Models from dir

When specifying a model like in

```yaml
- name: example-api-name
  kind: RealtimeAPI
  predictor:
    ...
    models:
      dir: s3://my-bucket/models/
```

the expected directory structure is

```text
  models/model-A/
  ├── model-A/
  |   └── * (predictor specific model files)
  └── model-B/
      ├── 24753823/
      |   └── * (predictor specific model files)
      └── 26234288/
          └── * (predictor specific model files)
```

Basically, when specifying the `predictor.models.dir` field, its value is expected to point to an S3 directory holding multiple models, each one either containing a model or versioned models.

## Model Caching

Model caching is a mechanism that builds on top of the [live model reloading](#live-model-reloading) mechanism.

Model caching is a mechanism that allows the API replica to access a vast number of models that wouldn't normally fit on an instance by only keeping a few in the memory at all times. This enables the user to specify thousands of models in the API spec while only running inferences on a couple of them.

This is useful when some models are frequently accessed whilst a larger portion of them are rarely used. It also helps manage costs by running the API on smaller-sized instances.

### Enabling model caching

Model caching is only available when the `predictor.models` field is specified. To enable model caching, values to `predictor.models.cache_size` and `predictor.models.disk_cache_size` fields have to be provided:

* `cache_size` represents the number of models to keep in memory.
* `disk_cache_size` represents the number of models to keep on disk. Must be equal or greater than `cache_size`.

With this modification in, nothing else has to be changed to the predictor's implementation. It's a drop-in functionality.

### Caveats

In the background, Cortex runs a special kind of garbage collector periodically (every 10 seconds) that looks in the memory/on-disk and checks if the number of models exceeds the threshold `cache_size`/`disk_cache_size` - if so, the least recently used models are evicted.

The limitation in this is that if lots of models are loaded in that 10 second timeframe (and the one after that and so on), then the steady state number of models in memory/on-disk can end up being higher than the specified threshold in the API spec config. This can potentially lead to OOM if too many are loaded. This is especially true when full scans across the whole number of models are continuously conducted.
