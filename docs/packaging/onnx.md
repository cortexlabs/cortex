# Packaging ONNX models

Export your trained model to the ONNX model format. Here is an example of an sklearn model being exported to ONNX:

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

<!-- CORTEX_VERSION_MINOR x4 -->
Here are complete examples of converting models from some of the common ML frameworks to ONNX:

* [XGBoost](https://colab.research.google.com/github/cortexlabs/cortex/blob/master/examples/xgboost/iris-classifier/xgboost.ipynb)

Upload your exported model to Amazon S3 using the AWS web console or CLI:

```bash
aws s3 cp model.onnx s3://my-bucket/model.onnx
```

Reference your model in an `api`:

```yaml
- kind: api
  name: my-api
  onnx:
    model: s3://my-bucket/model.onnx
```
