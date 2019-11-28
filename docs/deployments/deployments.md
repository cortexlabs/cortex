# Deployments

Deployments are used to group a set of APIs that are deployed together. It must be defined in every Cortex directory in a top-level `cortex.yaml` file.

## Configuration

```yaml
- kind: deployment
  name: <string>  # deployment name (required)
```

## Example

```yaml
- kind: deployment
  name: my_deployment
```

