# Python packages

## PyPI packages

You can install your required PyPI packages and import them in your Python files using pip. Cortex looks for
a `requirements.txt` file in the top level Cortex project directory (i.e. the directory which contains `cortex.yaml`):

```text
./my-classifier/
├── cortex.yaml
├── predictor.py
├── ...
└── requirements.txt
```

If you want to use `conda` to install your python packages, see the [Conda section](#conda-packages) below.

Note that some packages are pre-installed by default (see "pre-installed packages" for your Predictor type in the
Realtime API Predictor documentation and Batch API Predictor documentation).

## Private PyPI packages

To install packages from a private PyPI index, create a `pip.conf` inside the same directory as `requirements.txt`, and
add the following contents:

```text
[global]
extra-index-url = https://<username>:<password>@<my-private-index>.com/pip
```

In same directory, create a [`dependencies.sh` script](system-packages.md) and add the following:

```bash
cp pip.conf /etc/pip.conf
```

You may now add packages to `requirements.txt` which are found in the private index.

## GitHub packages

You can also install public/private packages from git registries (such as GitHub) by adding them to `requirements.txt`.
Here's an example for GitHub:

```text
# requirements.txt

# public access
git+https://github.com/<username>/<repo name>.git@<tag or branch name>#egg=<package name>

# private access
git+https://<personal access token>@github.com/<username>/<repo name>.git@<tag or branch name>#egg=<package name>
```

On GitHub, you can generate a personal access token by
following [these steps](https://help.github.com/en/github/authenticating-to-github/creating-a-personal-access-token-for-the-command-line)
.

## Installing with Setup

Python packages can also be installed by providing a `setup.py` that describes your project's modules. Here's an example
directory structure:

```text
./my-classifier/
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

## Conda packages

Cortex supports installing Conda packages. We recommend only using Conda when your required packages are not available
in PyPI. Cortex looks for a `conda-packages.txt` file in the top level Cortex project directory (i.e. the directory
which contains `cortex.yaml`):

```text
./my-classifier/
├── cortex.yaml
├── predictor.py
├── ...
└── conda-packages.txt
```

The `conda-packages.txt` file follows the format of `conda list --export`. Each line of `conda-packages.txt` should
follow this pattern: `[channel::]package[=version[=buildid]]`.

Here's an example of `conda-packages.txt`:

```text
conda-forge::rdkit
conda-forge::pygpu
```

In situations where both `requirements.txt` and `conda-packages.txt` are provided, Cortex installs Conda packages
in `conda-packages.txt` followed by PyPI packages in `requirements.txt`. Conda and Pip package managers install packages
and dependencies independently. You may run into situations where Conda and pip package managers install different
versions of the same package because they install and resolve dependencies independently from one another. To resolve
package version conflicts, it may be in your best interest to specify their exact versions in `conda-packages.txt`.

The current version of Python is `3.6.9`. Updating Python to a different version is possible with Conda, but there are
no guarantees that Cortex's web server will continue functioning correctly. If there's a change in Python's version, the
necessary core packages for the web server will be reinstalled. If you are using a custom base image, any other Python
packages that are built in to the image won't be accessible at runtime.

Check the [best practices](https://www.anaconda.com/using-pip-in-a-conda-environment/) on using `pip` inside `conda`.

## Customizing Dependency Paths

Cortex allows you to specify different dependency paths other than the default ones. This can be useful when deploying
different versions of the same API (e.g. CPU vs GPU dependencies).

To customize the path for your dependencies, you can specify `predictor.dependencies` in your API's configuration file. You can set
one or more fields to specify the path for each dependency type. Each path should be a relative path with respect to the current file.

For example:

```yaml
# cortex.yaml

- name: my-classifier
  kind: RealtimeAPI
  predictor:
    (...)
    dependencies:
      pip: requirement-gpu.txt
      conda: conda-packages-gpu.txt
      shell: dependencies-gpu.sh
```
