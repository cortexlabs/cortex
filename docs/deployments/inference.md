# Inference

Inference is a python file that describes how to intializes a model and use the model to make a prediction on a sample in a request.

The lifecycle of a replica running an inference implementation starts with loading the implementation and running any code living in the global scope. Once the implementation is loaded, Cortex calls the `init` function with metadata to do any additional preparations. The `init` function is typically used to download and initialize models because it receives a metadata object which normally contains the path to the exported/pickled model. Once the `init` function is executed, the replica is available to accept requests. The `predict` function is called when a request is recieved. The JSON payload of a request is parsed into a sample dictionary and is passed to the `predict` function along with metadata. The `predict` function is responsible for passing the sample into the model and returning a prediction.

Global variables can be used and shared across functions safely because each replica handles one request at a time.

## Implementation

```python
# Initialization code and variables can be declared here in global scope

def init(metadata):
    """Called once before the API is made available. Setup for model serving such as initializing the model or downloading vocabulary can be done here. Optional.

    Args:
        metadata: Custom dictionary specified by user in API configuration.
    """
    pass

def predict(sample, metadata):
    """Called once per request. Model prediction should be done here, including any preprocessing of the request payload and postprocessing of the model output. Required.

    Args:
        sample: A Python object parsed from a JSON request payload.
        metadata: Custom dictionary specified by user in API configuration.

    Returns:
        A prediction
    """
```

## Example

```python
from my_model import IrisNet
import boto3

labels = ["iris-setosa", "iris-versicolor", "iris-virginica"]

model = IrisNet() # Declare the model in global scope so that it can be used in init and prediction functions

def init(metadata):
    # Download model/model weights from S3 (location specified in api configuration metadata) and initialize your model.
    s3 = boto3.client("s3")
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

<!-- CORTEX_VERSION_MINOR -->
See [iris-classifier-pytorch](https://github.com/cortexlabs/cortex/blob/master/examples/iris-classifier-pytorch) for the full example.


## Pre-installed packages

The following packages have been pre-installed and can be used in your implementations:

```text
boto3==1.9.228
msgpack==0.6.1
numpy==1.17.2
requests==2.22.0
```

## PyPI packages

You can install additional PyPI packages and import them in your handlers. Cortex looks for a `requirements.txt` file in the top level Cortex project directory (i.e. the directory which contains `cortex.yaml`):

```text
./iris-classifier/
├── cortex.yaml
├── inference.py
├── ...
└── requirements.txt
```

## Project files

Cortex makes all files in the project directory (i.e. the directory which contains `cortex.yaml`) available to pre and post inference handlers. Python generated files, files and folders that start with `.`, and `cortex.yaml` are excluded.

The contents of the project directory is available in `/mnt/project/` in the API containers. For example, if this is your project directory:

```text
./iris-classifier/
├── cortex.yaml
├── config.json
├── inference.py
├── ...
└── requirements.txt
```

You can access `config.json` in `inference.py` like this:

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


```python
import ...
ctx = {}

def pre_inference(sample, metadata):
  ...
  ctx['tokenized_input'] = tokenized_input

def post_inference(prediction, metadata):
  tokenized_input = ctx['tokenized_input']
  ...
```
