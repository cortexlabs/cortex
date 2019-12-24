# TensorFlow

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

<!-- CORTEX_VERSION_MINOR -->
Export your trained model and upload the export directory, or a checkpoint directory containing the export directory (which is usually the case if you used `estimator.train_and_evaluate`). An example is shown below (here is the [complete example](https://github.com/cortexlabs/cortex/blob/master/examples/tensorflow/sentiment-analyzer)):

```python
import tensorflow as tf

...

OUTPUT_DIR="bert"
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

estimator.export_savedmodel(OUTPUT_DIR, serving_input_fn, strip_default_attrs=True)
```

Upload the checkpoint directory to Amazon S3 using the AWS web console or CLI:

```bash
aws s3 sync ./bert s3://my-bucket/bert
```

Reference your model in an `api`:

```yaml
- kind: api
  name: my-api
  predictor:
    type: tensorflow
    model: s3://my-bucket/bert
    path: predictor.py
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
  predictor:
    type: tensorflow
    model: s3://my-bucket/bert.zip
    path: predictor.py
```
