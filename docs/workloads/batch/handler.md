# Handler implementation

Batch APIs run distributed and fault-tolerant batch processing jobs on-demand. They can be used for batch inference or data processing workloads. It can also be used for running ML models using a variety of frameworks such as: PyTorch, ONNX, scikit-learn, XGBoost, TensorFlow (if not using `SavedModel`s), etc.

If you plan on deploying models with TensorFlow in `SavedModel` format and run inferences in batches, you can also use the [TensorFlow Handler](#models.md) that was specifically built for this purpose.

## Project files

Cortex makes all files in the project directory (i.e. the directory which contains `cortex.yaml`) available for use in your Handler class implementation. Python bytecode files (`*.pyc`, `*.pyo`, `*.pyd`), files or folders that start with `.`, and the api configuration file (e.g. `cortex.yaml`) are excluded.

The following files can also be added at the root of the project's directory:

* `.cortexignore` file, which follows the same syntax and behavior as a [.gitignore file](https://git-scm.com/docs/gitignore).
* `.env` file, which exports environment variables that can be used in the handler class. Each line of this file must follow the `VARIABLE=value` format.

For example, if your directory looks like this:

```text
./my-classifier/
├── cortex.yaml
├── values.json
├── predictor.py
├── ...
└── requirements.txt
```

You can access `values.json` in your Handler class like this:

```python
import json

class Handler:
    def __init__(self, config):
        with open('values.json', 'r') as values_file:
            values = json.load(values_file)
        self.values = values
```

## Interface

```python
# initialization code and variables can be declared here in global scope

class Handler:
    def __init__(self, config, job_spec):
        """(Required) Called once during each worker initialization. Performs
        setup such as downloading/initializing the model or downloading a
        vocabulary.

        Args:
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
        pass

    def handle_batch(self, payload, batch_id):
        """(Required) Called once per batch. Preprocesses the batch payload (if
        necessary), runs inference, postprocesses the inference output (if
        necessary), and writes the results to storage (i.e. S3 or a
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

## Structured logging

You can use Cortex's logger in your handler implemention to log in JSON. This will enrich your logs with Cortex's metadata, and you can add custom metadata to the logs by adding key value pairs to the `extra` key when using the logger. For example:

```python
...
from cortex_internal.lib.log import logger as cortex_logger

class Handler:
    def handle_batch(self, payload, batch_id):
        ...
        cortex_logger.info("completed processing batch", extra={"batch_id": batch_id, "confidence": confidence})
```

The dictionary passed in via the `extra` will be flattened by one level. e.g.

```text
{"asctime": "2021-01-19 15:14:05,291", "levelname": "INFO", "message": "completed processing batch", "process": 235, "batch_id": "iuasyd8f7", "confidence": 0.97}
```

To avoid overriding essential Cortex metadata, please refrain from specifying the following extra keys: `asctime`, `levelname`, `message`, `labels`, and `process`. Log lines greater than 5 MB in size will be ignored.

## Cortex Python client

A default [Cortex Python client](../../clients/python.md#cortex.client.client) environment has been configured for your API. This can be used for deploying/deleting/updating or submitting jobs to your running cluster based on the execution flow of your batch handler. For example:

```python
import cortex

class Handler:
    def on_job_complete(self):
        ...
        # get client pointing to the default environment
        client = cortex.client()
        # deploy API in the existing cluster using the artifacts in the previous step
        client.create_api(...)
```
