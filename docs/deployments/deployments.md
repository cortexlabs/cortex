# Deployments

Deployments are used to group a set of resources that can be deployed as a single unit. It must be defined in every Cortex directory in a top-level `cortex.yaml` file.

## Configuration

```yaml
- kind: deployment
  name: <string>  # deployment name (required)
  python_root: <string>  # path to the root of your python folder that will be appended to PYTHONPATH
```

## Example

```yaml
- kind: deployment
  name: my_deployment
```
