# External

Some Cortex resources support external importing.

## Model

You can serve a external model as an `API`.

1. Zip the exported estimator output in your checkpoint directory

Example:

```bash
ls export/estimator
saved_model.pb  variables/
zip -r model.zip export/estimator
```

2. Upload the zipped file to Amazon S3

Example:

```bash
aws s3 cp model.zip s3://your-bucket/model.zip
```

3. Specify `model_path` in an API kind

```yaml
- kind: api
  name: my-api
  model_path: s3://your-bucket/model.zip
  compute:
    replicas: 5
    gpu: 1
```
