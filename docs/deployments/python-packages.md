# Python packages

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

## Private packages on GitHub

You can also install private packages hosed on GitHub by adding them to `requirements.txt` using this syntax:

```text
# requirements.txt

git+https://<personal access token>@github.com/<username>/<repo name>.git@<tag or branch name>#egg=<package name>
```

You can generate a personal access token by following [these steps](https://help.github.com/en/github/authenticating-to-github/creating-a-personal-access-token-for-the-command-line).
