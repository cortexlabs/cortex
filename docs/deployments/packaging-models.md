# Packaging Models

## TensorFlow

Export your trained model and upload the export directory, or checkpoint directory containing the export directory, which is usually the case if you used `estimator.train_and_evaluate`. An example is shown below (here is the [complete example](https://github.com/cortexlabs/cortex/blob/master/examples/sentiment-analysis)):

```Python
import tensorflow as tf

...

OUPUT_DIR="bert"
estimator = tf.estimator.Estimator(model_fn=model_fn...)

# TF Serving requires a special input_fn used at serving time
def serving_input_fn():
    inputs = tf.placeholder(shape=[128], dtype=tf.int32)
    features = {
        "input_ids": tf.expand_dims(inputs, 0),
        "input_mask": tf.expand_dims(inputs, 0),
        "segment_ids": tf.expand_dims(inputs, 0),
        "label_ids": tf.placeholder(shape=[0], dtype=tf.int32),
    }
    return tf.estimator.export.ServingInputReceiver(features=features, receiver_tensors=inputs)

estimator.export_savedmodel(OUPUT_DIR, serving_input_fn, strip_default_attrs=True)
```

Upload the checkpoint directory to Amazon S3 using the AWS web console or CLI:

```bash
aws s3 sync ./bert s3://my-bucket/bert
```

Reference your model in an `api`:

```yaml
- kind: api
  name: my-api
  model: s3://my-bucket/bert
```

You may also zip the export directory before uploading it:

```bash
cd bert/1568244606  # Your version number will be different
zip -r bert.zip 1568244606
aws s3 cp bert.zip s3://my-bucket/bert.zip --profile prod
```

Reference the zipped model in an `api`:

```yaml
- kind: api
  name: my-api
  model: s3://my-bucket/bert.zip
```

## ONNX

Export your trained model to ONNX model format. An example of an sklearn model being exported to ONNX is shown below:

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

* [PyTorch](https://github.com/cortexlabs/cortex/blob/master/examples/iris-classifier/models/pytorch_model.py)
* [Sklearn](https://github.com/cortexlabs/cortex/blob/master/examples/iris-classifier/models/sklearn_model.py)
* [XGBoost](https://github.com/cortexlabs/cortex/blob/master/examples/iris-classifier/models/xgboost_model.py)
* [Keras](https://github.com/cortexlabs/cortex/blob/master/examples/iris-classifier/models/keras_model.py)

Upload your trained model in ONNX format to Amazon S3 using the AWS web console or CLI:

```bash
aws s3 cp model.onnx s3://my-bucket/model.onnx
```

Reference your model in an `api`:

```yaml
- kind: api
  name: my-api
  model: s3://my-bucket/model.onnx
```
