# APIs

Deploy your models as webservices at scale.

Specify a Python Predictor implementation that describes how to load your model and how to use it to make predictions. Cortex uses the Predictor to deploy multiple replicas that can serve your model as an API. Deployment parameters such as minimum replica count, maximum replica count, and prediction monitoring can be configured using YAML.

Besides providing a Predictor interface, Cortex can directly serve the following model formats:

- [TensorFlow saved model](./tensorflow-api.md)
- [ONNX](./onnx-api.md)

## Predictor

The Predictor interface consists of an `init` function and a `predict` function. The `init` function is reponsible for preparing the model for serving, downloading vocabulary files, aggregates etc. The `predict` function is called when a request received and is responsible for responding with a prediction.

```python
import ...

# variables declared in global scope can be used safely in both functions, one replica handles one request at a time
model = MyModel()
tokenizer = Tokenizer.init()
labels = requests.get('https://...')

def init(metadata):
  # download models and perform any additional setup here
  model_weight = download_weights_from_s3(metadata["model_path"])
  model.load(model_weight)

def predict(sample, metadata):
  # apply your model, preprocess the input and postprocess model output here
  tokens = tokenizer.encode(sample["text"])
  output = model(tokens)
  return labels[np.argmax(output)]
```

See [predictor](./predictor.md) for a detailed guide.

## Configuration

```yaml
- kind: api
  name: <string>  # API name (required)
  endpoint: <string>  # the endpoint for the API (default: /<deployment_name>/<api_name>)
  python_path: <string>  # path to the root of your python folder that will be appended to PYTHONPATH (default: folder containing cortex.yaml)
  python:
    predictor: <string>  # path to the inference implementation python file, relative to the cortex root (required)
  tracker:
    key: <string>  # key to track (required if the response payload is a JSON object)
    model_type: <string>  # model type, must be "classification" or "regression" (required)
  compute:
    min_replicas: <int>  # minimum number of replicas (default: 1)
    max_replicas: <int>  # maximum number of replicas (default: 100)
    init_replicas: <int>  # initial number of replicas (default: <min_replicas>)
    target_cpu_utilization: <int>  # CPU utilization threshold (as a percentage) to trigger scaling (default: 80)
    cpu: <string | int | float>  # CPU request per replica (default: 200m)
    gpu: <int>  # GPU request per replica (default: 0)
    mem: <string>  # memory request per replica (default: Null)
  metadata: <string: value>  # dictionary that can be used to configure custom values (optional)
```

### Example

```yaml
- kind: api
  name: my-api
  python:
    inference: inference.py
  compute:
    gpu: 1
```

## Debugging

You can log information about each request by adding a `?debug=true` parameter to your requests. This will print:

1. The raw sample
2. The value after running the `predict` function

## Prediction Monitoring

You can track your predictions by configuring a `tracker`. See [Prediction Monitoring](./prediction-monitoring.md) for more information.

## Autoscaling

Cortex automatically scales your webservices. See [Autoscaling](./autoscaling.md) for more information.
