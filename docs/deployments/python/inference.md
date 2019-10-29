# Inference

A python file that intializes a model and describes how to use the model to make predictions on data from JSON request payloads.

## Implementation

```python
def init(metadata):
    """Called once before API is made available. Setup for model serving such as initializing the model or downloading vocabulary should be done here.

    Args:
        metadata: Custom dictionary specified by user in API configuration.

    """
    pass

def predict(sample, metadata):
    """Called once per request. Model prediction including any preprocessing of request payload and postprocessing of model output should be done here.

    Args:
        sample: A python object parsed from a JSON payload of request.

        metadata: Custom dictionary specified by user in API configuration.

    Returns:
        A prediction
    """
```

## Example

```python
import numpy as np
import boto3
from my_models import MyNet

labels = ["iris-setosa", "iris-versicolor", "iris-virginica"]

model = MyNet()

def init(metadata):
    # Download model/model weights from S3 specified in your api metadata and initialize your model.
    s3 = boto3.client("s3", region_name=metadata["region"])
    s3.download_file(metadata["bucket"], metadata["key"], "iris_model.pth")
    model.load_state_dict(torch.load("iris_model.pth"))
    model.eval()


def predict(sample, metadata):
    # Convert dictionary of features passed from payload to tensor and pass it in to your model. Convert the model output to a label.
    input_tensor = torch.FloatTensor(
        [
            [
                sample["sepal_length"],
                sample["sepal_width"],
                sample["petal_length"],
                sample["petal_width"],
            ]
        ]
    )

    output = model(input_tensor)
    return labels[torch.argmax(output[0])]
```

See [iris-pytorch]() for full example.

## Pre-installed packages

The following packages have been pre-installed and can be used in your implementations:

```text
boto3==1.9.228
msgpack==0.6.1
numpy==1.17.2
requests==2.22.0
```

## PyPI packages

You can install additional PyPI packages and import them your handlers. Cortex looks for a `requirements.txt` file in the top level Cortex project directory (i.e. the directory which contains `cortex.yaml`):

```text
./iris-classifier/
├── cortex.yaml
├── inference.py
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
