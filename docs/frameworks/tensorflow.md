# TensorFlow and Keras Models

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

## TensorFlow Predictor

TensorFlow and Keras models can be exported in the SavedModel format and deployed using the TensorFlow Predictor. Please refer to the [TensorFlow Predictor documentation](../predictors/tensorflow.md).

<!-- CORTEX_VERSION_MINOR x2 -->
Most of the examples in [examples/tensorflow](https://github.com/cortexlabs/cortex/blob/master/examples/tensorflow) use the TensorFlow Predictor (e.g. [examples/tensorflow/iris-classifier](https://github.com/cortexlabs/cortex/blob/master/examples/tensorflow/iris-classifier), which includes a Python notebook demonstrating how its SavedModel was exported).

### Exporting a TensorFlow SavedModel

You may export your trained model and upload the export directory, or a checkpoint directory containing the export directory (which is usually the case if you used `estimator.train_and_evaluate()`). The folder may be zipped if desired.

<!-- CORTEX_VERSION_MINOR -->
Most of the TensorFlow examples use this approach. Here is the relevant code from [examples/tensorflow/sentiment-analyzer](https://github.com/cortexlabs/cortex/blob/master/examples/tensorflow/sentiment-analyzer):

```python
import tensorflow as tf

...

OUTPUT_DIR="bert"
estimator = tf.estimator.Estimator(model_fn=model_fn...)

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

The checkpoint directory is then uploaded to S3, e.g. via CLI:

```bash
aws s3 sync ./bert s3://cortex-examples/tensorflow/sentiment-analyzer/bert
```

And referenced in `cortex.yaml`:

```yaml
- name: sentiment-analyzer
  predictor:
    type: tensorflow
    path: predictor.py
    model: s3://cortex-examples/tensorflow/sentiment-analyzer/bert
```

<!-- CORTEX_VERSION_MINOR -->
The complete example can be found in [examples/tensorflow/sentiment-analyzer](https://github.com/cortexlabs/cortex/blob/master/examples/tensorflow/sentiment-analyzer).

It is also possible to zip the export directory before uploading it:

```bash
cd bert/1568244606  # Your version number will be different
zip -r bert.zip 1568244606
aws s3 cp bert.zip s3://my-bucket/bert.zip
```

You would then reference the zipped model in `cortex.yaml`:

```yaml
- name: sentiment-analyzer
  predictor:
    type: tensorflow
    path: predictor.py
    model: s3://my-bucket/bert.zip
```

## Python Predictor

TensorFlow and Keras models can also be deployed using the Python Predictor. Please refer to the [Python Predictor documentation](../predictors/python.md).

<!-- CORTEX_VERSION_MINOR -->
One Keras example that uses the Python Predictor is [examples/tensorflow/license-plate-reader](https://github.com/cortexlabs/cortex/blob/master/examples/tensorflow/license-plate-reader).
