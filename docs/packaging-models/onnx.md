# ONNX

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

Export your trained model to the ONNX model format. Here is an example of an sklearn model being exported to ONNX:

```python
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

Upload your exported model to Amazon S3 using the AWS web console or CLI:

```bash
aws s3 cp model.onnx s3://my-bucket/model.onnx
```

Reference your model in an `api`:

```yaml
- kind: api
  name: my-api
  predictor:
    type: onnx
    model: s3://my-bucket/model.onnx
    path: predictor.py
```
