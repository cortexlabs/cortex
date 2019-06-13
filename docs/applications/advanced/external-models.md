# Importing External Models

You can serve a model that was trained outside of Cortex as an API.

1. Zip the exported estimator output in your checkpoint directory, e.g.

```bash
$ ls export/estimator
saved_model.pb  variables/

$ zip -r model.zip export/estimator
```

2. Upload the zipped file to Amazon S3, e.g.

```bash
$ aws s3 cp model.zip s3://your-bucket/model.zip
```

3. Specify `external_model` in an API, e.g.

```yaml
- kind: api
  name: my-api
  external_model:
    path: s3://your-bucket/model.zip
    region: us-west-2
  compute:
    replicas: 5
    gpu: 1
```
