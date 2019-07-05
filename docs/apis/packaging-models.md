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
  model_format: tensorflow
  model: s3://my-bucket/model.zip
```

## ONNX

Export your trained model to an ONNX model format. An example of a trained model being exported to ONNX is shown below.

```Python
...

logreg_model = LogisticRegression(solver="lbfgs", multi_class="multinomial")

# Trained model
logreg_model.fit(X_train, y_train)

# Convert to ONNX model format
onnx_model = convert_sklearn(logreg_model, initial_types=[("input", FloatTensorType([1, 4]))])
with open("model.onnx", "wb") as f:
    f.write(onnx_model.SerializeToString())
```

See examples on how to convert models from common ML frameworks to ONNX.

* [PyTorch](https://github.com/cortexlabs/cortex/blob/master/examples/iris/models/pytorch.py)
* [Sklearn](https://github.com/cortexlabs/cortex/blob/master/examples/iris/models/sklearn.py)
* [XGBoost](https://github.com/cortexlabs/cortex/blob/master/examples/iris/models/xgboost.py)
* [Keras](https://github.com/cortexlabs/cortex/blob/master/examples/iris/models/keras.py)

Upload your trained model in ONNX format to S3

```text
$ aws s3 cp model.onnx s3://my-bucket/model.onnx
```

Specify `model` in an API, e.g.

```yaml
- kind: api
  name: my-api
  model_format: onnx
  model: s3://my-bucket/model.onnx
```
