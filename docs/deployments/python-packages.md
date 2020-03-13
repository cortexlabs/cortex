# Python packages

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

## PyPI packages

You can install your required PyPI packages and import them in your Python files. Cortex looks for a `requirements.txt` file in the top level Cortex project directory (i.e. the directory which contains `cortex.yaml`):

```text
./iris-classifier/
├── cortex.yaml
├── predictor.py
├── ...
└── requirements.txt
```

Note that some packages are pre-installed by default (see "pre-installed packages" for your Predictor type in the [Predictor documentation](predictors.md)).

## `setup.py`

It is also possible to reference Python libraries that are packaged using `setup.py`.

Here is an example directory structure:

```text
./iris-classifier/
├── cortex.yaml
├── predictor.py
├── ...
├── mypkg
│   └── __init__.py
├── requirements.txt
└── setup.py
```

In this case, `requirements.txt` can include a single `.`:

```python
# requirements.txt

.
```

If this is the contents `setup.py`:

```python
# setup.py

from distutils.core import setup

setup(
    name="mypkg",
    version="0.0.1",
    packages=["mypkg"],
)
```

And `__init__.py` looks like this:

```python
# mypkg/__init__.py

def hello():
    print("hello")
```

You can reference your package in `predictor.py`:

```python
# predictor.py

import mypkg

class PythonPredictor:
    def predict(self, payload):
        mypkg.hello()
```

## Private packages on GitHub

You can also install private packages hosed on GitHub by adding them to `requirements.txt` using this syntax:

```text
# requirements.txt

git+https://<personal access token>@github.com/<username>/<repo name>.git@<tag or branch name>#egg=<package name>
```

You can generate a personal access token by following [these steps](https://help.github.com/en/github/authenticating-to-github/creating-a-personal-access-token-for-the-command-line).
