# Predictor APIs

You can deploy models from any Python framework by defining a class definition that implements Cortex's Predictor interface. The class constructor is responsible for preparing the model for serving, downloading vocabulary files, etc. The `predict()` class function is called on every request and is responsible for responding with a prediction.

In addition to supporting Python models via the Predictor interface, Cortex can serve the following exported model formats:

- [TensorFlow](tensorflow.md)
- [ONNX](onnx.md)

## Configuration

```yaml
- kind: api
  name: <string>  # API name (required)
  endpoint: <string>  # the endpoint for the API (default: /<deployment_name>/<api_name>)
  predictor:
    path: <string>  # path to the predictor Python file, relative to the Cortex root (required)
    python_path: <string>  # path to the root of your Python folder that will be appended to PYTHONPATH (default: folder containing cortex.yaml)
    config: <string: value>  # dictionary of args passed to constructor of predictor (optional)
  tracker:
    key: <string>  # the JSON key in the response to track (required if the response payload is a JSON object)
    model_type: <string>  # model type, must be "classification" or "regression" (required)
  compute:
    min_replicas: <int>  # minimum number of replicas (default: 1)
    max_replicas: <int>  # maximum number of replicas (default: 100)
    init_replicas: <int>  # initial number of replicas (default: <min_replicas>)
    target_cpu_utilization: <int>  # CPU utilization threshold (as a percentage) to trigger scaling (default: 80)
    cpu: <string | int | float>  # CPU request per replica (default: 200m)
    gpu: <int>  # GPU request per replica (default: 0)
    mem: <string>  # memory request per replica (default: Null)
```

### Example

```yaml
- kind: api
  name: my-api
  predictor:
    path: predictor.py
  compute:
    gpu: 1
```

## Debugging

You can log information about each request by adding a `?debug=true` parameter to your requests. This will print:

1. The payload
2. The value after running the `predict` function

# Predictor

A Predictor is a Python class that describes how to initialize a model and use it to make a prediction.

The lifecycle of a replica using a Predictor starts with instantiating an instance of the Predictor class defined in the implementation file. The constructor of the Predictor class is typically used to download and initialize the model. It receives the config object, which is an arbitrary dictionary defined in the API configuration (it can be used to pass in the path to the exported/pickled model, vocabularies, aggregates, etc). After successfully initializing an instance of the Predictor class, the replica is available to accept requests. Upon receiving a request, the replica calls the `predict()` function with the JSON payload. The `predict()` function is responsible for returning a prediction from a sample.

## Implementation

```python
# initialization code and variables can be declared here in global scope

class Predictor:
    def __init__(self, config):
        """Called once before the API is made available. Setup for model serving such
        as downloading/initializing the model or downloading vocabulary can be done here.

        Args:
            config: Dictionary of args defined in API configuration, commonly contains path to model and other metadata.
        """
        pass

    def predict(self, payload):
        """Called once per request. Model prediction is done here, including any
        preprocessing of the request payload, batching of predictions and postprocessing of the model output.
        Required.

        Args:
            payload: The JSON request payload (parsed in Python).

        Returns:
            Predictions based on the request payload
        """
```

## Example

```python
import boto3
from my_model import IrisNet

class Predictor:
    def __init__(self, config):
        # Initialize the model
        model = IrisNet()
        model.load_state_dict(torch.load(model_path))
        model.eval()

        self.model = model
        self.labels = ["iris-setosa", "iris-versicolor", "iris-virginica"]


    def predict(self, payload):
        # Convert the request to a tensor and pass it into the model
        input_tensor = torch.FloatTensor(
            [
                [
                    payload["sepal_length"],
                    payload["sepal_width"],
                    payload["petal_length"],
                    payload["petal_width"],
                ]
            ]
        )

        # Run the prediction
        output = self.model(input_tensor)

        # Translate the model output to the corresponding label string
        return self.labels[torch.argmax(output[0])]
```

## Pre-installed packages

The following packages have been pre-installed and can be used in your implementations:

```text
boto3==1.10.13
cloudpickle==1.2.2
dill==0.3.1.1
joblib==0.14.0
Keras==2.3.1
msgpack==0.6.2
nltk==3.4.5
np-utils==0.5.11.1
numpy==1.17.3
pandas==0.25.3
Pillow==6.2.1
requests==2.22.0
scikit-image==0.16.2
scikit-learn==0.21.3
scipy==1.3.1
six==1.13.0
statsmodels==0.10.1
sympy==1.4
tensor2tensor==1.14.1
tensorflow-hub==0.7.0
tensorflow==2.0.0
torch==1.3.1
torchvision==0.4.2
xgboost==0.90
```

Learn how to install additional packages [here](../dependency-management/python-packages.md).
