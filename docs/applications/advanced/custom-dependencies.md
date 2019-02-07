# Custom Dependencies

Cortex allows you to customize the python libraries that can be used to build aggregators, transformers, and models.

## PyPI Packages

Cortex looks for a `requirements.txt` file in the top level directory of the app (in the same level as `app.yaml`). All packages listed in `requirements.txt` will be installed for your use in all aggregators, transformers, and models.

```text
./iris/
├── app.yaml
├── requirements.txt
├── samples.json
├── implementations/
└── resources/
```

## Custom Python Packages

Cortex can install your Python package(s) to be made available in all environments.

Cortex looks for your Python packages in the directory `./packages/<package name>`. The package must have a `setup.py` in the root of the package directory with the name set to your package name. Cortex will run `pip3 wheel -w wheelhouse ./packages/<package name>` to construct wheels for Python Project. These wheels will be installed for your use.

Cortex App Directory
```text
./iris/
├── app.yaml
├── samples.json
├── implementations/
├── resources/
└── packages
    └── acme-util
        ├── acme-util/
        |   ├── util_a.py
        |   └── util_b.py
        └── setup.py
```

Sample `setup.py`
```python
from distutils.core import setup

setup(
    name="acme-util",
    version="0.0.1",
    description="",
    author="ACME C. M. Exists",
    author_email="acme@acme.com",
    packages=["acme-util"],
    install_requires=["pyyaml==3.13"],
)
```