# Request Handlers

Request handlers are python files that can contain a `pre_inference` function and a `post_inference` function. Both functions are optional.

## Implementation

```python
def pre_inference(sample, metadata):
    """Prepare a sample before it is passed into the model.

    Args:
        sample: A sample from the request payload.

        metadata: Describes the expected shape and type of inputs to the model.
            If API model_format is tensorflow: map<string, SignatureDef>
                https://github.com/tensorflow/tensorflow/blob/master/tensorflow/core/protobuf/meta_graph.proto
            If API model_format is onnx: list<onnxruntime.NodeArg>
                https://microsoft.github.io/onnxruntime/api_summary.html#onnxruntime.NodeArg

    Returns:
        A dictionary containing model input names as keys and python lists or numpy arrays as values. If the model only has a single input, then a python list or numpy array can be returned.
    """
    pass

def post_inference(prediction, metadata):
    """Modify a prediction from the model before responding to the request.

    Args:
        prediction: The output of the model.

        metadata: Describes the output shape and type of outputs from the model.
            If API model_format is tensorflow: map<string, SignatureDef>
                https://github.com/tensorflow/tensorflow/blob/master/tensorflow/core/protobuf/meta_graph.proto
            If API model_format is onnx: list<onnxruntime.NodeArg>
                https://microsoft.github.io/onnxruntime/api_summary.html#onnxruntime.NodeArg

    Returns:
        A python dictionary or list.
    """
```

## Example

```python
import numpy as np

labels = ["iris-setosa", "iris-versicolor", "iris-virginica"]

def pre_inference(sample, metadata):
    # Convert a dictionary of features to a flattened in list in the order expected by the model
    return {
        metadata[0].name : [
            sample["sepal_length"],
            sample["sepal_width"],
            sample["petal_length"],
            sample["petal_width"],
        ]
    }


def post_inference(prediction, metadata):
    # Update the model prediction to include the index and the label of the predicted class
    probabilites = prediction[0][0]
    predicted_class_id = int(np.argmax(probabilites))
    return {
        "class_label": labels[predicted_class_id],
        "class_index": predicted_class_id,
        "probabilities": probabilites,
    }

```

## Pre-installed Packages

The following packages have been pre-installed and can be used in your implementations:

```text
boto3==1.9.78
msgpack==0.6.1
numpy>=1.13.3,<2
requirements-parser==0.2.0
packaging==19.0.0
pillow==6.1.0
regex==2017.4.5
requests==2.21.0
tensorflow==1.14.0  # In TensorFlow model formats only
```

You can install additional PyPI packages and import your own Python packages. See [Python Packages](../piplines/python-packages.md) for more details.

## Debugging

A Cortex logger can be imported and used in request handlers.

```
from cortex.lib.log import get_logger

logger = get_logger()

def pre_inference(sample, metadata):
    logger.info(sample)
    logger.info(metadata)
    ...
```

The output of these logs can be viewed using `cortex logs -v <api_name>`.
