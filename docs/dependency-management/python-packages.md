# Python packages

## PyPI packages

You can install your required PyPI packages and import them in your Python files. Cortex looks for a `requirements.txt` file in the top level Cortex project directory (i.e. the directory which contains `cortex.yaml`):

```text
./iris-classifier/
├── cortex.yaml
├── handler.py
├── ...
└── requirements.txt
```

Note that some packages are pre-installed by default (see [predictor](../deployments/predictor.md) or [request handlers](../deployments/request-handlers.md) depending on which runtime you're using).

## Private packages on GitHub

You can also install private packages hosed on GitHub by adding them to `requirements.txt` using this syntax:

```text
# requirements.txt

git+https://<personal access token>@github.com/<username>/<repo name>.git@<tag or branch name>#egg=<package name>
```

You can generate a personal access token by following [these steps](https://help.github.com/en/github/authenticating-to-github/creating-a-personal-access-token-for-the-command-line).

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

def pre_inference(payload, signature, metadata):
  print(config)
  ...
```
