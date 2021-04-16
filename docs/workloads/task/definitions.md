# Implementation

## Project files

Cortex makes all files in the project directory (i.e. the directory which contains `cortex.yaml`) available for use in your Task implementation. Python bytecode files (`*.pyc`, `*.pyo`, `*.pyd`), files or folders that start with `.`, and the api configuration file (e.g. `cortex.yaml`) are excluded.

The following files can also be added at the root of the project's directory:

* `.cortexignore` file, which follows the same syntax and behavior as a [.gitignore file](https://git-scm.com/docs/gitignore).
* `.env` file, which exports environment variables that can be used in the task. Each line of this file must follow the `VARIABLE=value` format.

For example, if your directory looks like this:

```text
./my-classifier/
├── cortex.yaml
├── values.json
├── task.py
├── ...
└── requirements.txt
```

You can access `values.json` in your Task like this:

```python
import json

class Task:
    def __call__(self, config):
        with open('values.json', 'r') as values_file:
            values = json.load(values_file)
        self.values = values
```

## Task

### Interface

```python
# initialization code and variables can be declared here in global scope

class Task:
    def __call__(self, config):
        """(Required) Task runnable.

        Args:
            config (required): Dictionary passed from API configuration (if
                specified) merged with configuration passed in with Job
                Submission API. If there are conflicting keys, values in
                configuration specified in Job submission takes precedence.
        """
        pass
```

## Structured logging

You can use Cortex's logger in your predictor implementation to log in JSON. This will enrich your logs with Cortex's metadata, and you can add custom metadata to the logs by adding key value pairs to the `extra` key when using the logger. For example:

```python
...
from cortex_internal.lib.log import logger as cortex_logger

class Task:
    def __call__(self, config):
        ...
        cortex_logger.info("completed validations", extra={"accuracy": accuracy})
```

The dictionary passed in via the `extra` will be flattened by one level. e.g.

```text
{"asctime": "2021-01-19 15:14:05,291", "levelname": "INFO", "message": "completed validations", "process": 235, "accuracy": 0.97}
```

To avoid overriding essential Cortex metadata, please refrain from specifying the following extra keys: `asctime`, `levelname`, `message`, `labels`, and `process`. Log lines greater than 5 MB in size will be ignored.

## Cortex Python client

A default [Cortex Python client](../../clients/python.md#cortex.client.client) environment has been configured for your API. This can be used for deploying/deleting/updating or submitting jobs to your running cluster based on the execution flow of your task. For example:

```python
import cortex

class Task:
    def __call__(self, config):
        ...
        # get client pointing to the default environment
        client = cortex.client()
        # deploy API in the existing cluster as part of your pipeline workflow
        client.create_api(...)
```
