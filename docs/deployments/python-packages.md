# Python/Conda packages

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

Within a deployment in Cortex, 2 package managers can be used to install additional python packages:

1. `pip`
1. `conda`

Both of these package managers have their uses and limitations. All default python packages on Cortex are installed with `pip`. It is therefore advised for any other add-on python package to be installed with `pip` and only resort to using `conda` when there are packages not available from PyPi or from any other index. The reason for this is to prevent inconsistencies within the virtual environment. Check the [best practices](https://www.anaconda.com/using-pip-in-a-conda-environment/) on using `pip` inside `conda`.

Keep in mind that pip installations come before the conda installations.

Note that some packages are pre-installed by default (see "pre-installed packages" for your Predictor type in the [Predictor documentation](predictors.md)).

*Note: The order of execution on all files is this: `script.sh` -> `.condarc` -> `environment.yaml` or `conda-packages.txt` -> `requirements.txt`.*

## Pip

With `pip`, packages can be installed from a few sources:

1. From [PyPi](https://pypi.org)'s index.

1. Locally, from the project's directory using `setup.py`.

1. From a git project (i.e. GitHub) that's either public or private.

### Installing from PyPi

You can install your required PyPI packages and import them in your Python files. Cortex looks for a `requirements.txt` file in the top level Cortex project directory (i.e. the directory which contains `cortex.yaml`):

```text
./iris-classifier/
├── cortex.yaml
├── predictor.py
├── ...
└── requirements.txt
```

### Installing with setup

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

### Installing from git

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

With `conda`, packages can be installed using 2 kinds of files, both mutually exclusive:

1. Using an `environment.yaml` config file.

1. Using a `conda-packages.txt` config file.

Cortex allows for the presence of a `.condarc` file in the root directory of the project. This can be used to tweak conda's configuration. Here's an example of `.condarc` that enables the pip-interoperabilty mechanism:

```text
pip_interop_enabled: false
```

More on `.condarc` can be found [here](https://docs.conda.io/projects/conda/en/latest/user-guide/configuration/use-condarc.html).

### Using environment.yaml

You can install packages using an `environment.yaml` config file using conda. Cortex looks for a `environment.yaml` file in the top level Cortex project directory. In the background, the command that's used against `environment.yaml` is `conda env update -n env --file environment.yaml`.

Here's an example of `environment.yaml` used to install `rdkit` python package:

```yaml
channels:
    - conda-forge
dependencies:
    - rdkit
```

`environment.yaml` can also be used to install `pip` packages and also introduce other indexes that may otherwise not be possible just by relying on `requirements.txt` alone. In addition to the previous example, here's one where `wget` is installed with pip:

```yaml
channels:
    - conda-forge
dependencies:
    - rdkit
    - pip
    - pip:
        - wget==3.2
```

Here's [an example from conda](https://github.com/conda/conda/blob/54e4a91d0da4d659a67e3097040764d3a2f6aa16/tests/conda_env/support/advanced-pip/environment.yml) that shows how far `enviroment.yaml` can be taken.

*Note: One word of advice is to not change the version of python deliberately because that can affect negatively the Cortex API. The current version of Python is `3.6.9`.*

### Using conda-packages.txt

You can install packages by providing a `conda-packages.txt` file to conda. Cortex looks for a `conda-packages.txt` file in the top level Cortex project directory. This file is executed by Cortex by running `conda install --file conda-packages.txt` command.

Here's an example of `conda-packages.txt` used to install `rdkit` and `pygpu` python packages:

```text
conda-forge::rdkit
conda-forge::pygpu
```
Each line inside `conda-packages.txt` follows the `[channel::]package[=version[=buildid]]` pattern.
