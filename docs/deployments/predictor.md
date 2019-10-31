# Predictor

Predictor is a Python file that describes how to initialize a model and use the model to make a prediction on a sample in a request.

The lifecycle of a replica running an Predictor starts with loading the implementation and executing code living in the global scope. Once the implementation is loaded, Cortex calls the `init` function with metadata to do any additional preparations. The `init` function is typically used to download and initialize models because it receives a metadata object which normally contains the path to the exported/pickled model. Once the `init` function is executed, the replica is available to accept requests. Upon receiving a request, the replica calls the `predict` function with a python dictionary of the JSON payload and the metadata object. The `predict` function is responsible for preprocessing input, applying the model, postprocessing the model output and responding with a prediction.

Global variables can be used and shared across functions safely because each replica handles one request at a time.

## Implementation

```python
# initialization code and variables can be declared here in global scope

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
import boto3
from my_model import IrisNet

labels = ["iris-setosa", "iris-versicolor", "iris-virginica"]
model = IrisNet()

def init(metadata):
    # Download model from S3 (location specified in api configuration metadata)
    s3 = boto3.client("s3")
    s3.download_file(metadata["bucket"], metadata["key"], "iris_model.pth")

    # Initialize the model
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
