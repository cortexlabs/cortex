# Application

The app resource is used to group a set of resources into an application that can be deployed as a singular unit. It must be defined in every Cortex application directory in a top-level `app.yaml` file.

## Config

```yaml
- kind: app  # (required)
  name: <string>  # app name (required)
```

## Example

```yaml
- kind: app
  name: my_app
```
