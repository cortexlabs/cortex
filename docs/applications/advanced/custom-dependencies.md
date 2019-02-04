# Custom Dependencies

Cortex allows you to customize the python libraries that can be used to build aggregators, transformers and models.

## PyPI Packages

Cortex looks for a requirements.txt file in the top level directory of the app (in the same level as app.yaml). All packages listed in requirements.txt will be installed for your use.

```text
./iris/
├── app.yaml
├── requirements.txt
├── samples.json
├── implementations/
└── resources/
```

## Your code

Cortex can install your Python package(s) to be made available in all environments.

Cortex looks for your Python packages in the directory `./packages/<package name>`. The package must have a `setup.py` in the root of the package directory with the name set to your package name. Cortex will run `pip3 wheel -w wheelhouse ./packages/<package name>` to construct wheels for Python Project. These wheels will be installed for your use.

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