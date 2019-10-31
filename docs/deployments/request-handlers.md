# Request handlers

Request handlers are useful for `tensorflow` or `onnx` exported models. They contain a `pre_inference` function and a `post_inference` function that can be used to process data before and after running an inference. Global variables can be used safely in both functions because each replica handles one request at a time.

## Implementation

```python
def pre_inference(sample, signature, metadata):
    """Prepare a sample before it is passed into the model.

    Args:
        sample: A Python object parsed from a JSON request payload.

        signature: Describes the expected shape and type of inputs to the model.
            If API model format is tensorflow: map<string, SignatureDef>
                https://github.com/tensorflow/tensorflow/blob/master/tensorflow/core/protobuf/meta_graph.proto
            If API model format is onnx: list<onnxruntime.NodeArg>
                https://microsoft.github.io/onnxruntime/api_summary.html#onnxruntime.NodeArg

        metadata: Custom dictionary specified by user in API configuration.

    Returns:
        A dictionary containing model input names as keys and python lists or numpy arrays as values. If the model only has a single input, then a python list or numpy array can be returned.
    """
    pass

def post_inference(prediction, signature, metadata):
    """Modify a prediction from the model before responding to the request.

    Args:
        prediction: The output of the model.

        signature: Describes the output shape and type of outputs from the model.
            If API model format is tensorflow: map<string, SignatureDef>
                https://github.com/tensorflow/tensorflow/blob/master/tensorflow/core/protobuf/meta_graph.proto
            If API model format is onnx: list<onnxruntime.NodeArg>
                https://microsoft.github.io/onnxruntime/api_summary.html#onnxruntime.NodeArg

        metadata: Custom dictionary specified by user in API configuration.

    Returns:
        A python dictionary or list.
    """
```

## Example

```python
import numpy as np

labels = ["iris-setosa", "iris-versicolor", "iris-virginica"]


# Convert the request payload to a flattened list
def pre_inference(sample, signature, metadata):
    return {
        signature[0].name: [
            sample["sepal_length"],
            sample["sepal_width"],
            sample["petal_length"],
            sample["petal_width"],
        ]
    }

# Respond with a string label
def post_inference(prediction, signature, metadata):
    predicted_class_id = prediction[0][0]
    return labels[predicted_class_id]
```
