# Python packages

## Pre-installed packages

The following packages have been pre-installed and can be used in your implementations:

```text
boto3==1.9.228
msgpack==0.6.1
numpy==1.17.2
requests==2.22.0
tensorflow==1.14.0  # For TensorFlow models only
onnxruntime==0.5.0  # For ONNX models only
```

## PyPI packages

You can install additional PyPI packages and import them in your Python files. Cortex looks for a `requirements.txt` file in the top level Cortex project directory (i.e. the directory which contains `cortex.yaml`):

```text
./iris-classifier/
├── cortex.yaml
├── handler.py
├── ...
└── requirements.txt
```

## Project files

Cortex makes all files in the project directory (i.e. the directory which contains `cortex.yaml`) available to request handlers. Python bytecode files (`*.pyc`, `*.pyo`, `*.pyd`), files or folders that start with `.`, and `cortex.yaml` are excluded.

The contents of the project directory is available in `/mnt/project/` in the API containers. For example, if this is your project directory:

```text
./iris-classifier/
├── cortex.yaml
├── config.json
├── handler.py
├── ...
└── requirements.txt
```

You can access `config.json` in `handler.py` like this:

```python
import json

with open('/mnt/project/config.json', 'r') as config_file:
  config = json.load(config_file)

def pre_inference(sample, signature, metadata):
  print(config)
  ...
```
