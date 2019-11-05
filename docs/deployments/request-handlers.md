# Request handlers

Request handlers can be used with TensorFlow or ONNX exported models. They contain a `pre_inference()` function and/or a `post_inference()` function that are to process data before and after running an inference. Global variables can be used safely in both functions because each replica handles one request at a time.

## Implementation

```python
def pre_inference(sample, signature, metadata):
    """Prepare a sample before it is passed into the model.

    Args:
        sample: The JSON request payload (parsed as a Python object).

        signature: Describes the expected shape and type of inputs to the model.
            If API model format is tensorflow: map<string, SignatureDef>
                https://github.com/tensorflow/tensorflow/blob/master/tensorflow/core/protobuf/meta_graph.proto
            If API model format is onnx: list<onnxruntime.NodeArg>
                https://microsoft.github.io/onnxruntime/api_summary.html#onnxruntime.NodeArg

        metadata: Custom dictionary specified by the user in API configuration.

    Returns:
        A dictionary containing model input names as keys and Python lists or numpy arrays as values. If the model only has a single input, then a Python list or numpy array can be returned.
    """
    pass

def post_inference(prediction, signature, metadata):
    """Modify model output before responding to the request.

    Args:
        prediction: The output of the model.

        signature: Describes the output shape and type of outputs from the model.
            If API model format is tensorflow: map<string, SignatureDef>
                https://github.com/tensorflow/tensorflow/blob/master/tensorflow/core/protobuf/meta_graph.proto
            If API model format is onnx: list<onnxruntime.NodeArg>
                https://microsoft.github.io/onnxruntime/api_summary.html#onnxruntime.NodeArg

        metadata: Custom dictionary specified by the user in API configuration.

    Returns:
        A Python dictionary or list.
    """
```

## Example

```python
import numpy as np

labels = ["iris-setosa", "iris-versicolor", "iris-virginica"]


def pre_inference(sample, signature, metadata):
    # Convert the request payload to a flattened list
    return {
        signature[0].name: [
            sample["sepal_length"],
            sample["sepal_width"],
            sample["petal_length"],
            sample["petal_width"],
        ]
    }

def post_inference(prediction, signature, metadata):
    # Respond with a string label
    predicted_class_id = prediction[0][0]
    return labels[predicted_class_id]
```

## Pre-installed packages

The following packages have been pre-installed and can be used in your implementations:

```text
boto3==1.9.228
msgpack==0.6.1
numpy==1.17.3
requests==2.22.0
dill==0.3.1.1
tensorflow==1.14.0  # For TensorFlow models only
onnxruntime==0.5.0  # For ONNX models only
```

Learn how to install additional packages [here](../dependencies/python-packages.md).
