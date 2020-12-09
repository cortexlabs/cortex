# Multi-model endpoints

<!-- CORTEX_VERSION_MINOR -->
It is possible to serve multiple models in the same Cortex API using any type of Cortex Predictor. In this guide we'll show the general outline of a multi-model deployment. The section for each predictor type is based on a corresponding example that can be found in the [examples directory](https://github.com/cortexlabs/cortex/tree/master/examples) of the Cortex project.

## Python Predictor

### Specifying models in API config

#### `cortex.yaml`

Even though it looks as if there's only a single model served, there are actually 4 different versions saved in `s3://cortex-examples/sklearn/mpg-estimator/linreg/`.

```yaml
- name: mpg-estimator
  kind: RealtimeAPI
  predictor:
    type: python
    path: predictor.py
    model_path: s3://cortex-examples/sklearn/mpg-estimator/linreg/
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

        # process the input
        # ...

        model = self.client.get_model(model_version=model_version)
        result = model.predict(model_input)

        return {"prediction": result, "model": {"version": model_version}}
```

#### Making predictions

For convenience, we'll export our API's endpoint (yours will be different from mine):

```bash
$ api_endpoint=http://a36473270de8b46e79a769850dd3372d-c67035afa37ef878.elb.us-west-2.amazonaws.com/mpg-estimator
```

Next, we'll make a prediction using the sentiment analyzer model by specifying the model version as a query parameter:

```bash
$ curl "${api_endpoint}?version=1" -X POST -H "Content-Type: application/json" -d @sample.json

{"prediction": 26.929889872154185, "model": {"version": "1"}}
```

Then we'll make a prediction using the 2nd version of the model (since they are just duplicate models, it will only return the same result):

```bash
$ curl "${api_endpoint}?version=2" -X POST -H "Content-Type: application/json" -d @sample.json

{"prediction": 26.929889872154185, "model": {"version": "2"}}
```

### Without specifying models in API config

For the Python Predictor, the API configuration for a multi-model API is similar to single-model APIs. The Predictor's `config` field can be used to customize the behavior of the `predictor.py` implementation.

#### `cortex.yaml`

```yaml
- name: multi-model-text-analyzer
  kind: RealtimeAPI
  predictor:
    type: python
    path: predictor.py
    config: {...}
    ...
```

#### `predictor.py`

Models should be loaded within the predictor's constructor. Query parameters are encouraged to be used when selecting the model for inference.

```python
# import modules here

class PythonPredictor:
    def __init__(self, config):
        # prepare the environment, download/load models/labels, etc
        # ...

        # load models
        self.analyzer = initialize_model("sentiment-analysis")
        self.summarizer = initialize_model("summarization")

    def predict(self, query_params, payload):
        # preprocessing
        model_name = query_params.get("model")
        model_input = payload["text"]
        # ...

        # make prediction
        if model_name == "sentiment":
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

#### Making predictions

For convenience, we'll export our API's endpoint (yours will be different from mine):

```bash
$ api_endpoint=http://a36473270de8b46e79a769850dd3372d-c67035afa37ef878.elb.us-west-2.amazonaws.com/multi-model-text-analyzer
```

Next, we'll make a prediction using the sentiment analyzer model by specifying the model name as a query parameter:

```bash
$ curl "${api_endpoint}?model=sentiment" -X POST -H "Content-Type: application/json" -d @sample-sentiment.json

{"label": "POSITIVE", "score": 0.9998506903648376}
```

Then we'll make a prediction using the text summarizer model:

```bash
$ curl "${api_endpoint}?model=summarizer" -X POST -H "Content-Type: application/json" -d @sample-summarizer.json

Machine learning is the study of algorithms and statistical models that computer systems use to perform a specific task. It is seen as a subset of artificial intelligence. Machine learning algorithms are used in a wide variety of applications, such as email filtering and computer vision. In its application across business problems, machine learning is also referred to as predictive analytics.
```

## TensorFlow Predictor

For the TensorFlow Predictor, a multi-model API is configured by placing the list of models in the Predictor's `models` field (each model will specify its own unique name). The `predict()` method of the `tensorflow_client` object expects a second argument that represents the name of the model that will be used for inference.

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
          model_path: s3://cortex-examples/tensorflow/image-classifier/inception/
        - name: iris
          model_path: s3://cortex-examples/tensorflow/iris-classifier/nn/
        - name: resnet50
          model_path: s3://cortex-examples/tensorflow/resnet50/
      ...
```

### `predictor.py`

```python
# import modules here

class TensorFlowPredictor:
    def __init__(self, tensorflow_client, config):
        # prepare the environment, download/load labels, etc
        # ...

        self.client = tensorflow_client

    def predict(self, payload, query_params):
        # preprocessing
        model_name = query_params["model"]
        model_input = preprocess(payload["url"])

        # make prediction
        results = self.client.predict(model_input, model_name)

        # postprocess
        predicted_label = postprocess(results)

        return {"label": predicted_label}
```

### Making predictions

For convenience, we'll export our API's endpoint (yours will be different from mine):

```bash
$ api_endpoint=http://a36473270de8b46e79a769850dd3372d-c67035afa37ef878.elb.us-west-2.amazonaws.com/multi-model-classifier
```

Next, we'll make a prediction using the iris classifier model by specifying the model name as a query parameter:

```bash
$ curl "${ENDPOINT}?model=iris" -X POST -H "Content-Type: application/json" -d @sample-iris.json

{"label": "setosa"}
```

Then we'll make a prediction using the resnet50 model:

```bash
$ curl "${ENDPOINT}?model=resnet50" -X POST -H "Content-Type: application/json" -d @sample-image.json

{"label": "sports_car"}
```

Finally we'll make a prediction using the inception model:

```bash
$ curl "${ENDPOINT}?model=inception" -X POST -H "Content-Type: application/json" -d @sample-image.json

{"label": "sports_car"}
```

## ONNX Predictor

For the ONNX Predictor, a multi-model API is configured by placing the list of models in the Predictor's `models` field (each model will specify its own unique name). The `predict()` method of the `onnx_client` object expects a second argument that represents the name of the model that will be used for inference.

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
          model_path: s3://cortex-examples/onnx/resnet50/
        - name: mobilenet
          model_path: s3://cortex-examples/onnx/mobilenet/
        - name: shufflenet
          model_path: s3://cortex-examples/onnx/shufflenet/
      ...
```

### `predictor.py`

```python
# import modules here

class ONNXPredictor:
    def __init__(self, onnx_client, config):
        # prepare the environment, download/load labels, etc
        # ...

        self.client = onnx_client

    def predict(self, payload, query_params):
        # process the input
        model_name = query_params["model"]
        model_input = preprocess(payload["url"])

        # make prediction
        results = self.client.predict(model_input, model_name)

        # postprocess
        predicted_label = postprocess(results)

        return {"label": predicted_label}

```

### Making predictions

For convenience, we'll export our API's endpoint (yours will be different from mine):

```bash
$ api_endpoint=http://a36473270de8b46e79a769850dd3372d-c67035afa37ef878.elb.us-west-2.amazonaws.com/multi-model-classifier
```

Next, we'll make a prediction using the resnet50 model by specifying the model name as a query parameter:

```bash
$ curl "${ENDPOINT}?model=resnet50" -X POST -H "Content-Type: application/json" -d @sample.json

{"label": "tabby"}
```

Then we'll make a prediction using the mobilenet model:

```bash
$ curl "${ENDPOINT}?model=mobilenet" -X POST -H "Content-Type: application/json" -d @sample.json

{"label": "tabby"}
```

Finally we'll make a prediction using the shufflenet model:

```bash
$ curl "${ENDPOINT}?model=shufflenet" -X POST -H "Content-Type: application/json" -d @sample.json

{"label": "Egyptian_cat"}
```
