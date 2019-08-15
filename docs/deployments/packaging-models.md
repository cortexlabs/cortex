# Packaging Models

## TensorFlow

Export your trained model and zip the model directory. An example is shown below (here is the [complete example](https://github.com/cortexlabs/cortex/blob/master/examples/iris/models/tensorflow_model.py)):

```Python
import tensorflow as tf
import shutil
import os

...

classifier = tf.estimator.Estimator(
    model_fn=my_model, model_dir="iris", params={"hidden_units": [10, 10], "n_classes": 3}
)

exporter = tf.estimator.FinalExporter("estimator", serving_input_fn, as_text=False)
train_spec = tf.estimator.TrainSpec(train_input_fn, max_steps=1000)
eval_spec = tf.estimator.EvalSpec(eval_input_fn, exporters=[exporter], name="estimator-eval")

tf.estimator.train_and_evaluate(classifier, train_spec, eval_spec)

# zip the estimator export dir (the exported path looks like iris/export/estimator/1562353043/)
shutil.make_archive("tensorflow", "zip", os.path.join("iris/export/estimator"))
```

Upload the zipped file to Amazon S3 using the AWS web console or CLI:

```text
$ aws s3 cp model.zip s3://my-bucket/model.zip
```

Reference your model in an `api`:

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

Here are complete examples of converting models from some of the common ML frameworks to ONNX:

* [PyTorch](https://github.com/cortexlabs/cortex/blob/master/examples/iris/models/pytorch_model.py)
* [Sklearn](https://github.com/cortexlabs/cortex/blob/master/examples/iris/models/sklearn_model.py)
* [XGBoost](https://github.com/cortexlabs/cortex/blob/master/examples/iris/models/xgboost_model.py)
* [Keras](https://github.com/cortexlabs/cortex/blob/master/examples/iris/models/keras_model.py)

Upload your trained model in ONNX format to Amazon S3 using the AWS web console or CLI:

```text
$ aws s3 cp model.onnx s3://my-bucket/model.onnx
```

Reference your model in an `api`:

```yaml
- kind: api
  name: my-api
  model: s3://my-bucket/model.onnx
```
