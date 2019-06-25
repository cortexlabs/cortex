# Importing Models

## TensorFlow

Zip the exported estimator output in your checkpoint directory, e.g.

```text
$ ls export/estimator
saved_model.pb  variables/

$ zip -r model.zip export/estimator
```

Upload the zipped file to Amazon S3, e.g.

```text
$ aws s3 cp model.zip s3://my-bucket/model.zip
```

Specify `model` in an API, e.g.

```yaml
- kind: api
  name: my-api
  external_model:
    path: s3://my-bucket/model.zip
    region: us-west-2
```
