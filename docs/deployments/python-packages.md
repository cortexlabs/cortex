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

## Conda packages

You can install Conda packages by creating a custom Docker image that first installs Conda and then installs your Conda packages.

Customize the template Dockerfile below with your desired Conda packages and follow these [instructions](./system-packages.md) to build and push your image to a container registry and configure Cortex to use your custom image.

```dockerfile
# Dockerfile

FROM <BASE CORTEX IMAGE>

# remove system-wide packages from the base image
RUN pip freeze > req && for pkg in "$(cat req)"; do pip uninstall $pkg -y || true; done && rm req

# add conda to path
ENV PATH /opt/conda/bin:$PATH

# install conda, it also includes py3.6.9
RUN curl https://repo.anaconda.com/miniconda/Miniconda3-4.7.12.1-Linux-x86_64.sh --output ~/miniconda.sh && \
    /bin/bash ~/miniconda.sh -b -p /opt/conda && \
    rm ~/miniconda.sh && \
    /opt/conda/bin/conda update conda && \
    /opt/conda/bin/conda install --force python=3.6.9 && \
    /opt/conda/bin/conda clean -tipsy && \
    ln -s /opt/conda/etc/profile.d/conda.sh /etc/profile.d/conda.sh && \
    echo ". /opt/conda/etc/profile.d/conda.sh" >> ~/.bashrc && \
    echo "conda activate base" >> ~/.bashrc

# install pip dependencies
RUN pip install --upgrade pip && \
    pip install --no-cache-dir -r /src/cortex/lib/requirements.txt && \
    pip install --no-cache-dir -r /src/cortex/serve/requirements.txt

# ---------------------------------------------------------- #
# Install your Conda packages here
# RUN conda install --no-update-deps -c conda-forge rdkit

# ---------------------------------------------------------- #

# replace system python with conda's version
RUN sed -i 's/\/usr\/bin\/python3.6/\/opt\/conda\/bin\/python/g' /src/cortex/serve/run.sh
```
