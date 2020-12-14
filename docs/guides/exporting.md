# Exporting models

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

Cortex can deploy models that are exported in a variety of formats. Therefore, it is best practice to export your model by following the recommendations of your machine learning library.

Here are examples for some common ML libraries:

## PyTorch

### `torch.save()`

The recommended approach is export your PyTorch model with [torch.save()](https://pytorch.org/docs/stable/torch.html?highlight=save#torch.save). Here is PyTorch's documentation on [saving and loading models](https://pytorch.org/tutorials/beginner/saving_loading_models.html). For example:

```python
torch.save(model.state_dict(), "weights.pth")
```

For Inferentia-equipped instances, check the [Inferentia instructions](inferentia.md#neuron).

### ONNX

It may also be possible to export your PyTorch model into the ONNX format using [torch.onnx.export()](https://pytorch.org/docs/stable/onnx.html#torch.onnx.export). For example:

```python
placeholder = torch.randn(1, 4)
torch.onnx.export(
    model, placeholder, "model.onnx", input_names=["input"], output_names=["species"]
)
```

For some ONNX exports, you may need to manually add a dynamic axis to represent batch size:

```python
import onnx

model = onnx.load('my_model.onnx')
model.graph.input[0].type.tensor_type.shape.dim[0].dim_param = '?'
onnx.save(model, 'my_model.onnx')
```

## TensorFlow / Keras

### `SavedModel`

You may export your trained model into an export directory, or use a checkpoint directory containing the export directory (which is usually the case if you used `estimator.train_and_evaluate()`). The folder may be zipped if you desire. For Inferentia-equipped instances, also check the [Inferentia instructions](inferentia.md#neuron).

A TensorFlow `SavedModel` directory should have this structure:

```text
1523423423/ (version prefix, usually a timestamp)
├── saved_model.pb
└── variables/
    ├── variables.index
    ├── variables.data-00000-of-00003
    ├── variables.data-00001-of-00003
    └── variables.data-00002-of-...
```

For example:

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

The checkpoint directory can then uploaded to S3, e.g. via the AWS CLI:

```bash
aws s3 sync ./bert s3://cortex-examples/tensorflow/sentiment-analyzer/bert
```

It is also possible to zip the export directory:

```bash
cd bert/1568244606  # Your version number will be different
zip -r bert.zip 1568244606
aws s3 cp bert.zip s3://my-bucket/bert.zip
```

### Other model formats

There are other ways to export Keras or TensorFlow models, and as long as they can be loaded and used to make predictions in Python, they will be supported by Cortex.

## Scikit-learn

### `pickle`

Scikit-learn models are typically exported using `pickle`. Here is [Scikit-learn's documentation](https://scikit-learn.org/stable/modules/model_persistence.html). For example:

```python
pickle.dump(model, open("model.pkl", "wb"))
```

### ONNX

It is also possible to export a scikit-learn model to the ONNX format using [onnxmltools](https://github.com/onnx/onnxmltools). For example:

```python
from sklearn.linear_model import LogisticRegression
from onnxmltools import convert_sklearn
from onnxconverter_common.data_types import FloatTensorType

...

logreg_model = LogisticRegression(solver="lbfgs", multi_class="multinomial")
logreg_model.fit(X_train, y_train)

# Export to ONNX model format
onnx_model = convert_sklearn(logreg_model, initial_types=[("input", FloatTensorType([1, 4]))])
with open("model.onnx", "wb") as f:
    f.write(onnx_model.SerializeToString())
```

## XGBoost

### `pickle`

XGBoost models can be exported using `pickle`.

For example:

```python
pickle.dump(model, open("model.pkl", "wb"))
```

### `Booster.save_model()`

XGBoost `Booster` models can also be exported using [`xgboost.Booster.save_model()`](https://xgboost.readthedocs.io/en/latest/python/python_api.html#xgboost.Booster.save_model). Auxiliary attributes of the Booster object (e.g. feature_names) will not be saved. To preserve all attributes, you can use `pickle` (see above).

For example:

```python
model.save_model("model.bin")
```

### ONNX

It is also possible to export an XGBoost model to the ONNX format using [onnxmltools](https://github.com/onnx/onnxmltools). For example:

```python
from onnxmltools.convert import convert_xgboost
from onnxconverter_common.data_types import FloatTensorType

onnx_model = convert_xgboost(model, initial_types=[("input", FloatTensorType([1, 4]))])
with open("gbtree.onnx", "wb") as f:
    f.write(onnx_model.SerializeToString())
```

## Other ML Libraries

Trained models should be exported by following the recommendations of the modeling framework you are using. `pickle` is commonly used, but some libraries have built-in functions for exporting models. It may also be possible to export your model to the ONNX format, e.g. using [onnxmltools](https://github.com/onnx/onnxmltools). As long as the exported model can be loaded and used to make predictions in Python, it will be supported by Cortex.
