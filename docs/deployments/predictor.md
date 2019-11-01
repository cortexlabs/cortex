# Predictor

Predictor is a Python file that describes how to initialize a model and use it to make a prediction on a sample in a request.

The lifecycle of a replica running a Predictor starts with loading the implementation file and executing code in the global scope. Once the implementation is loaded, Cortex calls the `init()` function to allow for any additional preparations. The `init()` function is typically used to download and initialize the model since it receives a metadata object, which is an arbitrary dictionary defined in the API config (it often contains the path to the exported/pickled model). Once the `init()` function is executed, the replica is available to accept requests. Upon receiving a request, the replica calls the `predict()` function with a Python dictionary of the JSON payload and the metadata object. The `predict()` function is responsible for preprocessing input, applying the model, postprocessing the model output, and returning a prediction.

Global variables can be shared across functions safely because each replica handles one request at a time.

## Implementation

```python
# initialization code and variables can be declared here in global scope

def init(metadata):
    """Called once before the API is made available. Setup for model serving such
    as downloading/initializing the model or downloading vocabulary can be done here.
    Optional.

    Args:
        metadata: Custom dictionary specified by the user in API configuration.
    """
    pass

def predict(sample, metadata):
    """Called once per request. Model prediction is done here, including any
    preprocessing of the request payload and postprocessing of the model output.
    Required.

    Args:
        sample: A Python object parsed from the JSON request payload.
        metadata: Custom dictionary specified by the user in API configuration.

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
    # Download model from S3 (location specified in the API's metadata)
    s3 = boto3.client("s3")
    s3.download_file(metadata["bucket"], metadata["key"], "iris_model.pth")

    # Initialize the model
    model.load_state_dict(torch.load("iris_model.pth"))
    model.eval()


def predict(sample, metadata):
    # Convert the request to a tensor and pass it into the model
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

    # Run the prediction
    output = model(input_tensor)

    # Convert the model output to a label
    return labels[torch.argmax(output[0])]
```
