# Python APIs

The Python Predictor implementation describes how to load your model and use it to make predictions. Cortex uses the Predictor to deploy your model as a web service. In addition to the Predictor interface, Cortex can serve the following exported model formats:

- [TensorFlow](tensorflow.md)
- [ONNX](onnx.md)

## Predictor

The Predictor interface consists of an `init()` function and a `predict()` function. The `init()` function is responsible for preparing the model for serving, downloading vocabulary files, aggregates etc. The `predict()` function is called on every request and is responsible for responding with a prediction.

```python
import ...

# variables declared in global scope can be used safely in both functions (one replica handles one request at a time)
model = MyModel()
tokenizer = Tokenizer.init()
labels = requests.get('https://...')

def init(metadata):
  # download models and perform any additional setup here
  model_weight = download_weights_from_s3(metadata["model_path"])
  model.load(model_weight)

def predict(sample, metadata):
  # process the input, apply your model, and postprocess model output here
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
  python_path: <string>  # path to the root of your Python folder that will be appended to PYTHONPATH (default: folder containing cortex.yaml)
  python:
    predictor: <string>  # path to the predictor Python file, relative to the Cortex root (required)
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
  metadata: <string: value>  # dictionary that can be used to configure custom values (optional)
```

### Example

```yaml
- kind: api
  name: my-api
  python:
    predictor: predictor.py
  compute:
    gpu: 1
```

## Debugging

You can log information about each request by adding a `?debug=true` parameter to your requests. This will print:

1. The raw sample
2. The value after running the `predict` function
