# Multi-model endpoints

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

<!-- CORTEX_VERSION_BRANCH_STABLE -->
It is possible to serve multiple models in the same Cortex API with all 3 kinds of predictors. In this guide we are showing you the general outline of a multi-model deployment. Each template is based on a corresponding example that can be found in the [examples directory](https://github.com/cortexlabs/cortex/tree/master/examples) of the Cortex project.

## Python Predictor

<!-- CORTEX_VERSION_BRANCH_STABLE -->
For the Python Predictor, we can expect a similar API configuration as that of a single-model deployment. The `predictor:config` field can be used to customize the behavior of the `predictor.py` implementation. This template is based on the [pytorch/multi-model-analyzer](https://github.com/cortexlabs/cortex/tree/master/examples/pytorch/multi-model-analyzer) example.

### `cortex.yaml`

```yaml
- name: text-analyzer
  predictor:
    type: python
    path: predictor.py
    config: ...
    ...
```

### `predictor.py`

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

            # postprocess
            predicted_label = postprocess(results)
            return {"label": predicted_label}
        elif model_name == "summarizer":
            results = self.summarizer(model_input)

            # postprocess
            predicted_label = postprocess(results)
            return {"label": predicted_label}
        else:
            return JSONResponse({"error": f"unknown model: {model_name}"}, status_code=400)
```

### Making predictions

To make predictions, you will first have to export your API's endpoint (yours will be different from mine):

```bash
$ api_endpoint=http://a36473270de8b46e79a769850dd3372d-c67035afa37ef878.elb.us-west-2.amazonaws.com/text-analyzer
```

Next, you'll make prediction requests on your input data whilst using a specified model as a query parameter. Here are 2 potential examples of making prediction requests.

```bash
$ curl ${api_endpoint}?model=sentiment -X POST -H "Content-Type: application/json" -d @sample-sentiment.json

{"label": "POSITIVE", "score": 0.9998506903648376}
```

```bash
$ curl ${api_endpoint}?model=summarizer -X POST -H "Content-Type: application/json" -d @sample-summarizer.json

Machine learning is the study of algorithms and statistical models that computer systems use to perform a specific task. It is seen as a subset of artificial intelligence. Machine learning algorithms are used in a wide variety of applications, such as email filtering and computer vision. In its application across business problems, machine learning is also referred to as predictive analytics.
```

## TensorFlow Predictor

<!-- CORTEX_VERSION_BRANCH_STABLE -->
For the TensorFlow Predictor, this template is based on the [tensorflow/multi-model-classifier](https://github.com/cortexlabs/cortex/tree/master/examples/tensorflow/multi-model-classifier) example. Models are placed in a list within the API's config using the `predictor:models` field. When the `predictor:models` field is filled in, the `predict` method of the TensorFlow client's object `tensorflow_client` expects a second argument `model_name` that represents the name of the model that will be used for inference. The models' names are specified using the `predictor:models:name` field.

### `cortex.yaml`

```yaml
- name: multi-model-classifier
  predictor:
    type: tensorflow
    path: predictor.py
    models:
      - name: iris
        model: s3://cortex-examples/tensorflow/iris-classifier/nn
      - name: inception
        model: s3://cortex-examples/tensorflow/image-classifier/inception
      - name: resnet50
        model: s3://cortex-examples/tensorflow/multi-model-classifier/resnet50
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
        # ...

        # make prediction
        results = self.client.predict(model_input, model_name)

        # postprocess
        predicted_label = postprocess(results)
        # ...
        return {"label": predicted_label}
```

### Making predictions

To make predictions, you will first have to export your API's endpoint (yours will be different from mine):

```bash
$ api_endpoint=http://a36473270de8b46e79a769850dd3372d-c67035afa37ef878.elb.us-west-2.amazonaws.com/multi-model-classifier
```

Next, you'll make prediction requests on your input data whilst using a specified model as a query parameter. Here are 3 potential examples of making prediction requests.

```bash
$ curl "${ENDPOINT}?model=iris" -X POST -H "Content-Type: application/json" -d @sample-iris.json

{"label": "setosa"}
```


```bash
$ curl "${ENDPOINT}?model=resnet50" -X POST -H "Content-Type: application/json" -d @sample-image.json

{"label": "sports_car"}
```

```bash
$ curl "${ENDPOINT}?model=inception" -X POST -H "Content-Type: application/json" -d @sample-image.json

{"label": "sports_car"}
```

## ONNX Predictor

<!-- CORTEX_VERSION_BRANCH_STABLE -->
For the ONNX Predictor, this template is based on the [onnx/multi-model-classifier](https://github.com/cortexlabs/cortex/tree/master/examples/onnx/multi-model-classifier) example. Models are placed in a list within the API's config using the `predictor:models` field. When the `predictor:models` field is filled in, the `predict` method of the ONNX client's object `onnx_client` expects a second argument `model_name` that represents the name of the model that will be used for inference. The models' names are specified using the `predictor:models:name` field.

### `cortex.yaml`

```yaml
- name: multi-model-classifier
  predictor:
    type: onnx
    path: predictor.py
    models:
      - name: resnet50
        model: s3://cortex-examples/onnx/resnet50/resnet50-v2-7.onnx
      - name: mobilenet
        model: s3://cortex-examples/onnx/mobilenet/mobilenetv2-7.onnx
      - name: shufflenet
        model: s3://cortex-examples/onnx/shufflenet/shufflenet-v2-10.onnx
      ...
```

### `predictor.py`

```python
# import modules here

class ONNXPredictor:
    def __init__(self, onnx_client, config):
        # prepare the environment, download/load labels, etc
        # ...

        # onnx client
        self.client = onnx_client

    def predict(self, payload, query_params):
        # process the input
        model_name = query_params["model"]
        model_input = preprocess(payload["url"])
        # ...

        # make prediction
        results = self.client.predict(model_input, model_name)

        # postprocess
        predicted_label = postprocess(results)
        # ...
        return {"label": predicted_label}

```

### Making predictions

To make predictions, you will first have to export your API's endpoint (yours will be different from mine):

```bash
$ api_endpoint=http://a36473270de8b46e79a769850dd3372d-c67035afa37ef878.elb.us-west-2.amazonaws.com/multi-model-classifier
```

Next, you'll make prediction requests on your input data whilst using a specified model as a query parameter. Here are 3 potential examples of making prediction requests.

```bash
curl "${ENDPOINT}?model=resnet50" -X POST -H "Content-Type: application/json" -d @sample.json

{"label": "tabby"}
```

```bash
curl "${ENDPOINT}?model=mobilenet" -X POST -H "Content-Type: application/json" -d @sample.json

{"label": "tabby"}
```

```bash
curl "${ENDPOINT}?model=shufflenet" -X POST -H "Content-Type: application/json" -d @sample.json

{"label": "Egyptian_cat"}
```
