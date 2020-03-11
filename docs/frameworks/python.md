# Scikit-learn, XGBoost, and other Python-based frameworks

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

## Python Predictor

Models trained using Python-based frameworks such as scikit-learn and XGBoost can be deployed using the Python Predictor. Please refer to the [Python Predictor documentation](../predictors/python.md).

<!-- CORTEX_VERSION_MINOR -->
The examples in [examples/sklearn](https://github.com/cortexlabs/cortex/blob/master/examples/sklearn) all use the Python Predictor.

### Exporting Python models

Trained models should be exported by following the recommendations of the modeling framework you are using. Once exported, the model can be loaded in the Predictor's `__init__()` function.

For example, the scikit-learn iris classifier example exports the model using `pickle`:

```python
pickle.dump(model, open("model.pkl", "wb"))
```

`model.pkl` is then uploaded to S3 and referenced in `cortex.yaml`:

```yaml
- name: iris-classifier
  predictor:
    type: python
    path: predictor.py
    config:
      bucket: cortex-examples
      key: sklearn/iris-classifier/model.pkl
```

And the model is loaded in `predictor.py`:

```python
s3.download_file(config["bucket"], config["key"], "model.pkl")
self.model = pickle.load(open("model.pkl", "rb"))
```

<!-- CORTEX_VERSION_MINOR -->
The complete example can be found in [examples/sklearn/iris-classifier](https://github.com/cortexlabs/cortex/blob/master/examples/sklearn/iris-classifier).

## ONNX Predictor

Alternatively, it is often possible to export a model into the ONNX format and deploy the exported model using the ONNX Predictor. Please refer to the [ONNX Predictor documentation](../predictors/onnx.md).

<!-- CORTEX_VERSION_MINOR -->
One example which uses the ONNX Predictor is [examples/xgboost/iris-classifier](https://github.com/cortexlabs/cortex/blob/master/examples/xgboost/iris-classifier).

### Exporting to ONNX

Depending on your modeling framework, you may be able to export your model to ONNX using [onnxmltools](https://github.com/onnx/onnxmltools).

For example, the XGBoost iris classifier example exports the model like this:

```python
from onnxmltools.convert import convert_xgboost
from onnxconverter_common.data_types import FloatTensorType

onnx_model = convert_xgboost(xgb_model, initial_types=[("input", FloatTensorType([1, 4]))])
with open("gbtree.onnx", "wb") as f:
    f.write(onnx_model.SerializeToString())
```

`gbtree.onnx` is then uploaded to S3 and referenced in `cortex.yaml`:

```yaml
- name: iris-classifier
  predictor:
    type: onnx
    path: predictor.py
    model: s3://cortex-examples/xgboost/iris-classifier/gbtree.onnx
```

<!-- CORTEX_VERSION_MINOR -->
The complete example can be found in [examples/xgboost/iris-classifier](https://github.com/cortexlabs/cortex/blob/master/examples/xgboost/iris-classifier).

Similarly, here is an example of exporting a scikit-learn model to ONNX:

```python
from sklearn.linear_model import LogisticRegression
from onnxmltools import convert_sklearn
from onnxconverter_common.data_types import FloatTensorType

...

logreg_model = LogisticRegression(solver="lbfgs", multi_class="multinomial")
logreg_model.fit(X_train, y_train)

# Convert to ONNX model format
onnx_model = convert_sklearn(logreg_model, initial_types=[("input", FloatTensorType([1, 4]))])
with open("model.onnx", "wb") as f:
    f.write(onnx_model.SerializeToString())
```

You would then upload `model.onnx` to S3 and reference your model in `cortex.yaml`:

```yaml
- name: my-api
  predictor:
    type: onnx
    model: s3://my-bucket/model.onnx
    path: predictor.py
```
