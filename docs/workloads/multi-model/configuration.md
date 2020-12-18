# Configuration

## `PythonPredictor`

### Specifying models in API configuration

#### `cortex.yaml`

The directory `s3://cortex-examples/sklearn/mpg-estimator/linreg/` contains 4 different versions of the model.

```yaml
- name: mpg-estimator
  kind: RealtimeAPI
  predictor:
    type: python
    path: predictor.py
    models:
      path: s3://cortex-examples/sklearn/mpg-estimator/linreg/
```

#### `predictor.py`

```python
import mlflow.sklearn
import numpy as np


class PythonPredictor:
    def __init__(self, config, python_client):
        self.client = python_client

    def load_model(self, model_path):
        return mlflow.sklearn.load_model(model_path)

    def predict(self, payload, query_params):
        model_version = query_params.get("version")

        # model_input = ...

        model = self.client.get_model(model_version=model_version)
        result = model.predict(model_input)

        return {"prediction": result, "model": {"version": model_version}}
```

### Without specifying models in API configuration

#### `cortex.yaml`

```yaml
- name: text-analyzer
  kind: RealtimeAPI
  predictor:
    type: python
    path: predictor.py
    ...
```

#### `predictor.py`

```python
class PythonPredictor:
    def __init__(self, config):
        self.analyzer = initialize_model("sentiment-analysis")
        self.summarizer = initialize_model("summarization")

    def predict(self, query_params, payload):
        model_name = query_params.get("model")
        model_input = payload["text"]

        # ...

        if model_name == "analyzer":
            results = self.analyzer(model_input)
            predicted_label = postprocess(results)
            return {"label": predicted_label}
        elif model_name == "summarizer":
            results = self.summarizer(model_input)
            predicted_label = postprocess(results)
            return {"label": predicted_label}
        else:
            return JSONResponse({"error": f"unknown model: {model_name}"}, status_code=400)
```

## `TensorFlowPredictor`

### `cortex.yaml`

```yaml
- name: multi-model-classifier
  kind: RealtimeAPI
  predictor:
    type: tensorflow
    path: predictor.py
    models:
      paths:
        - name: inception
          path: s3://cortex-examples/tensorflow/image-classifier/inception/
        - name: iris
          path: s3://cortex-examples/tensorflow/iris-classifier/nn/
        - name: resnet50
          path: s3://cortex-examples/tensorflow/resnet50/
      ...
```

### `predictor.py`

```python
class TensorFlowPredictor:
    def __init__(self, tensorflow_client, config):
        self.client = tensorflow_client

    def predict(self, payload, query_params):
        model_name = query_params["model"]
        model_input = preprocess(payload["url"])
        results = self.client.predict(model_input, model_name)
        predicted_label = postprocess(results)
        return {"label": predicted_label}
```

## `ONNXPredictor`

### `cortex.yaml`

```yaml
- name: multi-model-classifier
  kind: RealtimeAPI
  predictor:
    type: onnx
    path: predictor.py
    models:
      paths:
        - name: resnet50
          path: s3://cortex-examples/onnx/resnet50/
        - name: mobilenet
          path: s3://cortex-examples/onnx/mobilenet/
        - name: shufflenet
          path: s3://cortex-examples/onnx/shufflenet/
      ...
```

### `predictor.py`

```python
class ONNXPredictor:
    def __init__(self, onnx_client, config):
        self.client = onnx_client

    def predict(self, payload, query_params):
        model_name = query_params["model"]
        model_input = preprocess(payload["url"])
        results = self.client.predict(model_input, model_name)
        predicted_label = postprocess(results)
        return {"label": predicted_label}
```
