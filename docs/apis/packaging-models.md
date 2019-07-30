# Packaging Models

## TensorFlow

Zip the exported estimator output in your checkpoint directory:

```text
$ ls export/estimator/1560263597/
saved_model.pb  variables/

$ zip -r model.zip export/estimator
```

Upload the zipped file to Amazon S3:

```text
$ aws s3 cp model.zip s3://my-bucket/model.zip
```

Reference your `model` in an API:

```yaml
- kind: api
  name: my-api
  model: s3://my-bucket/model.zip
```

## ONNX

Export your trained model to an ONNX model format. An example of an sklearn model being exported to ONNX is shown below:

```Python
from sklearn.linear_model import LogisticRegression
from onnxmltools import convert_sklearn
from onnxconverter_common.data_types import FloatTensorType

...

logreg_model = LogisticRegression(solver="lbfgs", multi_class="multinomial")
logreg_model.fit(X_train, y_train)

# Convert to ONNX model format
onnx_model = convert_sklearn(logreg_model, initial_types=[("input", FloatTensorType([1, 4]))])
with open("sklearn.onnx", "wb") as f:
    f.write(onnx_model.SerializeToString())
```

Here are examples of converting models from some of the common ML frameworks to ONNX:

* [PyTorch](https://github.com/cortexlabs/cortex/blob/master/examples/iris/models/pytorch_model.py)
* [Sklearn](https://github.com/cortexlabs/cortex/blob/master/examples/iris/models/sklearn_model.py)
* [XGBoost](https://github.com/cortexlabs/cortex/blob/master/examples/iris/models/xgboost_model.py)
* [Keras](https://github.com/cortexlabs/cortex/blob/master/examples/iris/models/keras_model.py)

Upload your trained model in ONNX format to Amazon S3:

```text
$ aws s3 cp model.onnx s3://my-bucket/model.onnx
```

Reference your `model` in an API:

```yaml
- kind: api
  name: my-api
  model: s3://my-bucket/model.onnx
```
