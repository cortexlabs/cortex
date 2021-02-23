# System packages

Cortex looks for a file named `dependencies.sh` in the top level Cortex project directory (i.e. the directory which contains `cortex.yaml`). For example:

```text
./my-classifier/
├── cortex.yaml
├── predictor.py
├── ...
└── dependencies.sh
```

`dependencies.sh` is executed with `bash` shell during the initialization of each replica (before installing Python packages in `requirements.txt` or `conda-packages.txt`). Typical use cases include installing required system packages to be used in your Predictor, building Python packages from source, etc. If initialization time is a concern, see [Docker images](images.md) for how to build and use custom Docker images.

Here is an example `dependencies.sh`, which installs the `tree` utility:

```bash
apt-get update && apt-get install -y tree
```

The `tree` utility can now be called inside your `predictor.py`:

```python
# predictor.py
import subprocess

class PythonPredictor:
    def __init__(self, config):
        subprocess.run(["tree"])
    ...
```

If you need to upgrade the Python Runtime version on your image, you can do so in your `dependencies.sh` file:

```bash
# upgrade python runtime version
conda update -n base -c defaults conda
conda install -n env python=3.8.5

# re-install cortex core dependencies
/usr/local/cortex/install-core-dependencies.sh
```

## Customizing Dependency Paths

Cortex allows you to specify a path for this script other than `dependencies.sh`. This can be useful when deploying
different versions of the same API (e.g. CPU vs GPU dependencies). The path should be a relative path with respect
to the API configuration file, and is specified via `predictor.dependencies.shell`.

For example:

```yaml
# cortex.yaml

- name: my-classifier
  kind: RealtimeAPI
  predictor:
    (...)
    dependencies:
      shell: dependencies-gpu.sh
```
