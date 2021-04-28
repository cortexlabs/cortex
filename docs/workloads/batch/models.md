# TensorFlow Models

In addition to the [standard Python Handler](handler.md), Cortex also supports another handler called the TensorFlow handler, which can be used to run TensorFlow models exported as `SavedModel` models.

## Interface

**Uses TensorFlow version 2.3.0 by default**

```python
class Handler:
    def __init__(self, tensorflow_client, config, job_spec):
        """(Required) Called once during each worker initialization. Performs
        setup such as downloading/initializing the model or downloading a
        vocabulary.

        Args:
            tensorflow_client (required): TensorFlow client which is used to
                make predictions. This should be saved for use in handle_batch().
            config (required): Dictionary passed from API configuration (if
                specified) merged with configuration passed in with Job
                Submission API. If there are conflicting keys, values in
                configuration specified in Job submission takes precedence.
            job_spec (optional): Dictionary containing the following fields:
                "job_id": A unique ID for this job
                "api_name": The name of this batch API
                "config": The config that was provided in the job submission
                "workers": The number of workers for this job
                "total_batch_count": The total number of batches in this job
                "start_time": The time that this job started
        """
        self.client = tensorflow_client
        # Additional initialization may be done here

    def handle_batch(self, payload, batch_id):
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
Cortex provides a `tensorflow_client` to your Handler class' constructor. `tensorflow_client` is an instance of [TensorFlowClient](https://github.com/cortexlabs/cortex/tree/master/pkg/cortex/serve/cortex_internal/lib/client/tensorflow.py) that manages a connection to a TensorFlow Serving container to make predictions using your model. It should be saved as an instance variable in your Handler class, and your `handle_batch()` function should call `tensorflow_client.predict()` to make an inference with your exported TensorFlow model. Preprocessing of the JSON payload and postprocessing of predictions can be implemented in your `handle_batch()` function as well.

When multiple models are defined using the Handler's `models` field, the `tensorflow_client.predict()` method expects a second argument `model_name` which must hold the name of the model that you want to use for inference (for example: `self.client.predict(payload, "text-generator")`). There is also an optional third argument to specify the model version.

If you need to share files between your handler implementation and the TensorFlow Serving container, you can create a new directory within `/mnt` (e.g. `/mnt/user`) and write files to it. The entire `/mnt` directory is shared between containers, but do not write to any of the directories in `/mnt` that already exist (they are used internally by Cortex).

## `predict` method

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

## Specifying models

Whenever a model path is specified in an API configuration file, it should be a path to an S3 prefix which contains your exported model. Directories may include a single model, or multiple folders each with a single model (note that a "single model" need not be a single file; there can be multiple files for a single model). When multiple folders are used, the folder names must be integer values, and will be interpreted as the model version. Model versions can be any integer, but are typically integer timestamps. It is always assumed that the highest version number is the latest version of your model.

### API spec

#### Single model

The most common pattern is to serve a single model per API. The path to the model is specified in the `path` field in the `handler.models` configuration. For example:

```yaml
# cortex.yaml

- name: iris-classifier
  kind: BatchAPI
  handler:
    # ...
    type: tensorflow
    models:
      path: s3://my-bucket/models/text-generator/
```

#### Multiple models

It is possible to serve multiple models from a single API. The paths to the models are specified in the api configuration, either via the `models.paths` or `models.dir` field in the `handler` configuration. For example:

```yaml
# cortex.yaml

- name: iris-classifier
  kind: BatchAPI
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
  kind: BatchAPI
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

### Structure

#### On CPU/GPU

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

#### On Inferentia

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
