# Python



## Implementation

```python
def init(sample, signature, metadata):
    """Initialize models 

    Args:
        sample: A sample from the request payload.

        metadata: Custom dictionary specified by user in API configuration.

    """
    pass

def post_inference(prediction, signature, metadata):
    """Modify a prediction from the model before responding to the request.

    Args:
        sample: A sample from the request payload.

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

Cortex makes all files in the project directory (i.e. the directory which contains `cortex.yaml`) available to pre and post inference handlers. Python generated files and files and folders that start with `.` are excluded.

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
