# Request handlers

Request handlers are python files that can contain a `pre_inference` function and a `post_inference` function. Both functions are optional. Request handlers can be provided for `tensorflow` or `onnx` exported models. 

Global variables can be used and shared across functions safely because each replica handles one request at a time.

## Implementation

```python
# variables declared in global scope can be used safely in both functions, one replica handles one request at a time

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

def pre_inference(sample, signature, metadata):
    # Convert a dictionary of features to a flattened in list in the order expected by the model
    return {
        signature[0].name : [
            sample["sepal_length"],
            sample["sepal_width"],
            sample["petal_length"],
            sample["petal_width"],
        ]
    }


def post_inference(prediction, signature, metadata):
    # Update the model prediction to include the index and the label of the predicted class
    probabilites = prediction[0][0]
    predicted_class_id = int(np.argmax(probabilites))
    return {
        "class_label": labels[predicted_class_id],
        "class_index": predicted_class_id,
        "probabilities": probabilites,
    }
```

## Pre-installed packages

The following packages have been pre-installed and can be used in your implementations:

```text
boto3==1.9.228
msgpack==0.6.1
numpy==1.17.2
requests==2.22.0
tensorflow==1.14.0  # For TensorFlow models only
onnxruntime==0.5.0  # For ONNX models only
```

## PyPI packages

You can install additional PyPI packages and import them your handlers. Cortex looks for a `requirements.txt` file in the top level Cortex project directory (i.e. the directory which contains `cortex.yaml`):

```text
./iris-classifier/
├── cortex.yaml
├── handler.py
├── ...
└── requirements.txt
```

## Project files

Cortex makes all files in the project directory (i.e. the directory which contains `cortex.yaml`) available to pre and post inference handlers. Python generated files and files, folders that start with `.`, and `cortex.yaml` are excluded.

The contents of the project directory is available in `/mnt/project/` in the API containers. For example, if this is your project directory:

```text
./iris-classifier/
├── cortex.yaml
├── config.json
├── handler.py
├── ...
└── requirements.txt
```

You can access `config.json` in `handler.py` like this:

```python
import json

with open('/mnt/project/config.json', 'r') as config_file:
  config = json.load(config_file)

def pre_inference(sample, signature, metadata):
  print(config)
  ...
```

## System packages

You can also install additional system packages. See [System Packages](system-packages.md) for more details.
