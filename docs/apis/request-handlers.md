# Request Handlers

## Implementation

```python
def preinference(sample, metadata):
    """Prepare a sample before it is passed into the model.

    Args:
        sample: A single sample in the request payload converted from JSON to Python object.

        metadata: Describes the expected shape and type of inputs to the model.
            If API model_type is tensorflow the object is a map<string, SignatureDef>
                https://github.com/tensorflow/tensorflow/blob/master/tensorflow/core/protobuf/meta_graph.proto
            If API model_type is onnx the object is a list of [onnxruntime.NodeArg]
                https://microsoft.github.io/onnxruntime/api_summary.html#onnxruntime.NodeArg

    Returns:
        If model only has one 1 input, return a python list or numpy array of expected type  and shape. If model has more than 1 input, return a dictionary mapping input names to python list or numpy array of expected type and shape.
    """
    pass

def postinference(prediction, metadata):
    """Modify prediction from model before adding it to response payload.

    Args:
        sample: A single sample in the request payload converted from JSON to Python object

        metadata: Describes the output shape and type of outputs from the model.
            If API model_type is tensorflow the object is a map<string, SignatureDef>
                https://github.com/tensorflow/tensorflow/blob/master/tensorflow/core/protobuf/meta_graph.proto
            If API model_type is onnx the object is a list of [onnxruntime.NodeArg]
                https://microsoft.github.io/onnxruntime/api_summary.html#onnxruntime.NodeArg

    Returns:
        Python object that can be marshalled to JSON.
    """
```

## Example

```python
import numpy as np

iris_labels = ["Iris-setosa", "Iris-versicolor", "Iris-virginica"]

def preinference(request, metadata):
    return {
        metadata[0].name : [
            request["sepal_length"],
            request["sepal_width"],
            request["petal_length"],
            request["petal_width"],
        ]
    }


def postinference(response, metadata):
    predicted_class_id = response[0][0]
    return {"class_label": iris_labels[predicted_class_id], "class_index": predicted_class_id}

```

## Pre-installed Packages

The following packages have been pre-installed and can be used in your implementations:

```text
boto3==1.9.78
msgpack==0.6.1
numpy>=1.13.3,<2
requirements-parser==0.2.0
packaging==19.0.0
```

You can install additional PyPI packages and import your own Python packages. See [Python Packages](../advanced/python-packages.md) for more details.
