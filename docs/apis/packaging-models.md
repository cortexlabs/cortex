# Packaging Models

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
  model_type: tensorflow
  model: s3://my-bucket/model.zip
```

## ONNX

Convert your model to ONNX model format. 

```Python
# Convert PyTorch model to ONNX
dummy_input = torch.randn(1, 4)

torch.onnx.export(
    model, dummy_input, "iris_pytorch.onnx", input_names=["input"], output_names=["species"]
)
```

See examples on how to convert models from common ML frameworks to ONNX.

* [PyTorch](https://github.com/cortexlabs/cortex/blob/master/examples/models/iris_pytorch.py)
* [scikit-learn](https://github.com/cortexlabs/cortex/blob/master/examples/models/iris_sklearn_logreg.py)
* [XGBoost](https://github.com/cortexlabs/cortex/blob/master/examples/models/iris_xgboost.py)
* [Keras](https://github.com/cortexlabs/cortex/blob/master/examples/models/iris_keras.py)

```text
$ aws s3 cp model.onnx s3://my-bucket/model.onnx
```

Specify `model` in an API, e.g.

```yaml
- kind: api
  name: my-api
  model_type: onnx
  model: s3://my-bucket/model.onnx
```
