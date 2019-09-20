# Python Packages

Cortex allows you to install additional Python packages to be used in any workload. If your Python packages require system packages to be installed, see [system packages](system-packages.md) for more details.

## PyPI Packages

Cortex looks for a `requirements.txt` file in the top level cortex directory (in the same level as `cortex.yaml`).

```text
./iris-classifier/
├── cortex.yaml
├── ...
└── requirements.txt
```
