# Models

Live model reloading is a mechanism that periodically checks for updated models in the model path(s) provided in `handler.models`. It is automatically enabled for all handler types, including the Python handler type (as long as model paths are specified via `multi_model_reloading` in the `handler` configuration).

The following is a list of events that will trigger the API to update its model(s):

* A new model is added to the model directory.
* A model is removed from the model directory.
* A model changes its directory structure.
* A file in the model directory is updated in-place.

## Python Handler

To use live model reloading with the Python handler, the model path(s) must be specified in the API's `handler` configuration, via the `multi_model_reloading` field. When models are specified in this manner, your `Handler` class must implement the `load_model()` function, and models can be retrieved by using the `get_model()` method of the `model_client` that's passed into your handler's constructor.

### Example

```python
class Handler:
    def __init__(self, config, model_client):
        self.client = model_client

    def load_model(self, model_path):
        # model_path is a path to your model's directory on disk
        return load_from_disk(model_path)

    def handle_post(self, payload):
      model = self.client.get_model()
      return model.predict(payload)
```

When multiple models are being served in an API, `model_client.get_model()` can accept a model name:

```python
class Handler:
    # ...

    def handle_post(self, payload, query_params):
      model = self.client.get_model(query_params["model"])
      return model.predict(payload)
```

`model_client.get_model()` can also accept a model version if a version other than the highest is desired:

```python
class Handler:
    # ...

    def handle_post(self, payload, query_params):
      model = self.client.get_model(query_params["model"], query_params["version"])
      return model.predict(payload)
```

### Interface

```python
# initialization code and variables can be declared here in global scope

class Handler:
    def __init__(self, config, model_client):
        """(Required) Called once before the API becomes available. Performs
        setup such as downloading/initializing the model or downloading a
        vocabulary.

        Args:
            config (required): Dictionary passed from API configuration (if
                specified). This may contain information on where to download
                the model and/or metadata.
            model_client (required): Python client which is used to retrieve
                models for prediction. This should be saved for use in the handler method.
                Required when `handler.multi_model_reloading` is specified in
                the api configuration.
        """
        self.client = model_client

    def load_model(self, model_path):
        """Called by Cortex to load a model when necessary.

        This method is required when `handler.multi_model_reloading`
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

    # define any handler methods for HTTP/gRPC workloads here
```

<!-- CORTEX_VERSION_MINOR -->
When explicit model paths are specified in the Python handler's API configuration, Cortex provides a `model_client` to your Handler's constructor. `model_client` is an instance of [ModelClient](https://github.com/cortexlabs/cortex/tree/0.35/python/serve/cortex_internal/lib/client/python.py) that is used to load model(s) (it calls the `load_model()` method of your handler, which must be defined when using explicit model paths). It should be saved as an instance variable in your handler class, and your handler method should call `model_client.get_model()` to load your model for inference. Preprocessing of the JSON/gRPC payload and postprocessing of predictions can be implemented in your handler method as well.

When multiple models are defined using the Handler's `multi_model_reloading` field, the `model_client.get_model()` method expects an argument `model_name` which must hold the name of the model that you want to load (for example: `self.client.get_model("text-generator")`). There is also an optional second argument to specify the model version.

### `load_model` method

The `load_model()` method that you implement in your `Handler` can return anything that you need to make a prediction. There is one caveat: whatever the return value is, it must be unloadable from memory via the `del` keyword. The following frameworks have been tested to work:

* PyTorch (CPU & GPU)
* ONNX (CPU & GPU)
* Sklearn/MLFlow (CPU)
* Numpy (CPU)
* Pandas (CPU)
* Caffe (not tested, but should work on CPU & GPU)

Python data structures containing these types are also supported (e.g. lists and dicts).

The `load_model()` method takes a single argument, which is a path (on disk) to the model to be loaded. Your `load_model()` method is called behind the scenes by Cortex when you call the `model_client`'s `get_model()` method. Cortex is responsible for downloading your model from S3 onto the local disk before calling `load_model()` with the local path. Whatever `load_model()` returns will be the exact return value of `model_client.get_model()`. Here is the schema for `model_client.get_model()`:

```python
def get_model(model_name, model_version):
    """
    Retrieve a model for inference.

    Args:
        model_name (optional): Name of the model to retrieve (when multiple models are deployed in an API).
            When handler.models.paths is specified, model_name should be the name of one of the models listed in the API config.
            When handler.models.dir is specified, model_name should be the name of a top-level directory in the models dir.
        model_version (string, optional): Version of the model to retrieve. Can be omitted or set to "latest" to select the highest version.

    Returns:
        The value that's returned by your handler's load_model() method.
    """
```

### Specifying models

Whenever a model path is specified in an API configuration file, it should be a path to an S3 prefix which contains your exported model. Directories may include a single model, or multiple folders each with a single model (note that a "single model" need not be a single file; there can be multiple files for a single model). When multiple folders are used, the folder names must be integer values, and will be interpreted as the model version. Model versions can be any integer, but are typically integer timestamps. It is always assumed that the highest version number is the latest version of your model.

#### API spec

##### Single model

The most common pattern is to serve a single model per API. The path to the model is specified in the `path` field in the `handler.multi_model_reloading` configuration. For example:

```yaml
# cortex.yaml

- name: iris-classifier
  kind: RealtimeAPI
  handler:
    # ...
    type: python
    multi_model_reloading:
      path: s3://my-bucket/models/text-generator/
```

##### Multiple models

It is possible to serve multiple models from a single API. The paths to the models are specified in the api configuration, either via the `multi_model_reloading.paths` or `multi_model_reloading.dir` field in the `handler` configuration. For example:

```yaml
# cortex.yaml

- name: iris-classifier
  kind: RealtimeAPI
  handler:
    # ...
    type: python
    multi_model_reloading:
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
  handler:
    # ...
    type: python
    multi_model_reloading:
      dir: s3://my-bucket/models/
```

It is also not necessary to specify the `multi_model_reloading` section at all, since you can download and load the model in your handler's `__init__()` function. That said, it is necessary to use the `multi_model_reloading` field to take advantage of live model reloading or multi-model caching.

When using the `multi_model_reloading.paths` field, each path must be a valid model directory (see above for valid model directory structures).

When using the `multi_model_reloading.dir` field, the directory provided may contain multiple subdirectories, each of which is a valid model directory. For example:

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

#### Structure

Any model structure is accepted. Here is an example:

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

## TensorFlow Handler

In addition to the [standard Python Handler](handler.md), Cortex also supports another handler called the TensorFlow handler, which can be used to run TensorFlow models exported as `SavedModel` models. When using the TensorFlow handler, the model path(s) must be specified in the API's `handler` configuration, via the `models` field.

### Example

```python
class Handler:
    def __init__(self, tensorflow_client, config):
        self.client = tensorflow_client

    def handle_post(self, payload):
      return self.client.predict(payload)
```

When multiple models are being served in an API, `tensorflow_client.predict()` can accept a model name:

```python
class Handler:
    # ...

    def handle_post(self, payload, query_params):
      return self.client.predict(payload, query_params["model"])
```

`tensorflow_client.predict()` can also accept a model version if a version other than the highest is desired:

```python
class Handler:
    # ...

    def handle_post(self, payload, query_params):
      return self.client.predict(payload, query_params["model"], query_params["version"])
```

Note: when using Inferentia models with the TensorFlow handler type, live model reloading is only supported if `handler.processes_per_replica` is set to 1 (the default value).

### Interface

```python
class Handler:
    def __init__(self, tensorflow_client, config):
        """(Required) Called once before the API becomes available. Performs
        setup such as downloading/initializing a vocabulary.

        Args:
            tensorflow_client (required): TensorFlow client which is used to
                make predictions. This should be saved for use in the handler method.
            config (required): Dictionary passed from API configuration (if
                specified).
        """
        self.client = tensorflow_client
        # Additional initialization may be done here

    # define any handler methods for HTTP/gRPC workloads here
```

<!-- CORTEX_VERSION_MINOR -->
Cortex provides a `tensorflow_client` to your Handler's constructor. `tensorflow_client` is an instance of [TensorFlowClient](https://github.com/cortexlabs/cortex/tree/0.35/python/serve/cortex_internal/lib/client/tensorflow.py) that manages a connection to a TensorFlow Serving container to make predictions using your model. It should be saved as an instance variable in your Handler class, and your handler method should call `tensorflow_client.predict()` to make an inference with your exported TensorFlow model. Preprocessing of the JSON payload and postprocessing of predictions can be implemented in your handler method as well.

When multiple models are defined using the Handler's `models` field, the `tensorflow_client.predict()` method expects a second argument `model_name` which must hold the name of the model that you want to use for inference (for example: `self.client.predict(payload, "text-generator")`). There is also an optional third argument to specify the model version.

If you need to share files between your handler implementation and the TensorFlow Serving container, you can create a new directory within `/mnt` (e.g. `/mnt/user`) and write files to it. The entire `/mnt` directory is shared between containers, but do not write to any of the directories in `/mnt` that already exist (they are used internally by Cortex).

### `predict` method

Inference is performed by using the `predict` method of the `tensorflow_client` that's passed to the handler's constructor:

```python
def predict(model_input, model_name, model_version) -> dict:
    """
    Run prediction.

    Args:
        model_input: Input to the model.
        model_name (optional): Name of the model to retrieve (when multiple models are deployed in an API).
            When handler.models.paths is specified, model_name should be the name of one of the models listed in the API config.
            When handler.models.dir is specified, model_name should be the name of a top-level directory in the models dir.
        model_version (string, optional): Version of the model to retrieve. Can be omitted or set to "latest" to select the highest version.

    Returns:
        dict: TensorFlow Serving response converted to a dictionary.
    """
```

### Specifying models

Whenever a model path is specified in an API configuration file, it should be a path to an S3 prefix which contains your exported model. Directories may include a single model, or multiple folders each with a single model (note that a "single model" need not be a single file; there can be multiple files for a single model). When multiple folders are used, the folder names must be integer values, and will be interpreted as the model version. Model versions can be any integer, but are typically integer timestamps. It is always assumed that the highest version number is the latest version of your model.

#### API spec

##### Single model

The most common pattern is to serve a single model per API. The path to the model is specified in the `path` field in the `handler.models` configuration. For example:

```yaml
# cortex.yaml

- name: iris-classifier
  kind: RealtimeAPI
  handler:
    # ...
    type: tensorflow
    models:
      path: s3://my-bucket/models/text-generator/
```

##### Multiple models

It is possible to serve multiple models from a single API. The paths to the models are specified in the api configuration, either via the `models.paths` or `models.dir` field in the `handler` configuration. For example:

```yaml
# cortex.yaml

- name: iris-classifier
  kind: RealtimeAPI
  handler:
    # ...
    type: tensorflow
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
  handler:
    # ...
    type: tensorflow
    models:
      dir: s3://my-bucket/models/
```

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

#### Structure

##### On CPU/GPU

The model path must be a SavedModel export:

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

##### On Inferentia

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
