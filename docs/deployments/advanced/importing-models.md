# Importing Models

## TensorFlow

1. Zip the exported estimator output in your checkpoint directory, e.g.

```bash
$ ls export/estimator
saved_model.pb  variables/

$ zip -r model.zip export/estimator
```

1. Upload the zipped file to Amazon S3, e.g.

```bash
$ aws s3 cp model.zip s3://your-bucket/model.zip
```

1. Specify `model` in an API, e.g.

```yaml
- kind: api
  name: my-api
  model: s3://my-bucket/model.zip
```
