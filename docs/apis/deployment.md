# Deployment

The deployment resource is used to group a set of APIs that can be deployed as a single unit. It must be defined in every Cortex directory in a top-level `cortex.yaml` file.

## Config

```yaml
- kind: deployment
  name: <string>  # deployment name (required)
```

## Example

```yaml
- kind: deployment
  name: my_deployment
```
