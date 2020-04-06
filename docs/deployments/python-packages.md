# Python/Conda packages

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

## PyPI packages

You can install your required PyPI packages and import them in your Python files using pip. Cortex looks for a `requirements.txt` file in the top level Cortex project directory (i.e. the directory which contains `cortex.yaml`):

```text
./iris-classifier/
├── cortex.yaml
├── predictor.py
├── ...
└── requirements.txt
```

If you want to use `conda` to install your python packages, see the [Conda section](#conda) below.

Note that some packages are pre-installed by default (see "pre-installed packages" for your Predictor type in the [Predictor documentation](predictors.md)).

## Installing with Setup

Python packages can also be installed by providing a `setup.py` that describes your project's modules. Here's an example directory structure:

```text
./iris-classifier/
├── cortex.yaml
├── predictor.py
├── ...
├── mypkg
│   └── __init__.py
├── requirements.txt
└── setup.py
```

In this case, `requirements.txt` will have this form:
```text
# requirements.txt

.
```

## Installing from GitHub

You can also install public/private packages from git registries (such as GitHub) by adding them to `requirements.txt`. Here's an example for GitHub:

```text
# requirements.txt

# public access
git+https://github.com/<username>/<repo name>.git@<tag or branch name>#egg=<package name>

# private access
git+https://<personal access token>@github.com/<username>/<repo name>.git@<tag or branch name>#egg=<package name>
```

On GitHub, you can generate a personal access token by following [these steps](https://help.github.com/en/github/authenticating-to-github/creating-a-personal-access-token-for-the-command-line).

## Conda

Cortex supports installing Conda packages. We recommend only using Conda when your required packages are not available in PyPI. Cortex looks for a `conda-packages.txt` file in the top level Cortex project directory (i.e. the directory which contains `cortex.yaml`):

```text
./iris-classifier/
├── cortex.yaml
├── predictor.py
├── ...
└── conda-packages.txt
```

The `conda-packages.txt` file follows the format of `conda list --export`. Each line of `conda-packages.txt` should follow this pattern: `[channel::]package[=version[=buildid]]`.

Here's an example of `conda-packages.txt`:
```text
conda-forge::rdkit
conda-forge::pygpu
```

In situations where both `requirements.txt` and `conda-packages.txt` are provided, Cortex installs Conda packages in `conda-packages.txt` followed by PyPI packages in `requirements.txt`. Conda and Pip package managers install packages and dependencies independently. You may run into situations where Conda and pip package managers install different versions of the same package because they install and resolve dependencies independently from one another. To resolve package version conflicts, it may be in your best interest to specify their exact versions in `conda-packages.txt`.

Check the [best practices](https://www.anaconda.com/using-pip-in-a-conda-environment/) on using `pip` inside `conda`.
