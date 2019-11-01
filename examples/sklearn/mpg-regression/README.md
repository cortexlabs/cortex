# Deploy a Sklearn Linear Regression model

This example shows how to deploy a sklearn linear regression model trained on the MPG dataset.

## Predictor

We implement Cortex's Python Predictor interface that describes how to load the model and make predictions using the model. Cortex will use this implementation to serve your model as an API of autoscaling replicas. We specify a `requirements.txt` to install dependencies necessary to implement the Cortex Predictor interface.

### Initialization

Cortex executes the python implementation and calls the `init` function once on startup. We download the model from S3 based on the location specified in the `metadata` key of our deployment configuration.
```python
model = None

def init(metadata):
    global model
    s3 = boto3.client("s3")
    s3.download_file(metadata["bucket"], metadata["key"], "mpg.joblib")
    model = load("mpg.joblib")
```

### Predict
The `predict` function will be triggered once per request to run the text generation model on a prompt provided in the request. In the `predict` function, we extract the features from the sample sent in the request and respond with a predicted mpg.

```python
def predict(sample, metadata):
    arr = [
        sample["cylinders"],
        sample["displacement"],
        sample["horsepower"],
        sample["weight"],
        sample["acceleration"],
    ]
    result = model.predict([arr])
    return np.asscalar(result)
```

See [predictor.py](./src/predictor.py) for the complete code.

## Define a deployment

A `deployment` specifies a set of resources that are deployed as a single unit. An `api` makes the Cortex python implementation available as a web service that can serve real-time predictions. The metadata specified in this configuration will be passed into the `init` function in `predictor.py` for model initialization. Once the model is initialized the `predict` function in `predictor.py` will be triggered every time a request is made to the API.

```yaml
- kind: deployment
  name: auto

- kind: api
  name: mpg
  python:
    predictor: src/predictor.py
  tracker:
    model_type: regression
  metadata:
    bucket: cortex-examples
    key: mpg-regression/mpg.joblib
```

## Deploy to AWS

`cortex deploy` takes the declarative configuration from `cortex.yaml` and creates it on the cluster.

```bash
$ cortex deploy

deployment started
```

Behind the scenes, Cortex containerizes the Predictor implementation, makes it servable using Flask, exposes the endpoint with a load balancer, and orchestrates the workload on Kubernetes.

You can track the status of a deployment using `cortex get`:

```bash
$ cortex get mpg --watch

status   up-to-date   available   requested   last update   avg latency
live     1            1           1           9m            -
```

The output above indicates that one replica of the API was requested and one replica is available to serve predictions. Cortex will automatically launch more replicas if the load increases and spin down replicas if there is unused capacity.

## Serve real-time predictions

```bash
$ cortex get mpg

url: http://***.amazonaws.com/auto/mpg

$ curl http://***.amazonaws.com/auto/mpg \
    -X POST -H "Content-Type: application/json" \
    -d '{"cylinders": 4, "displacement": 135, "horsepower": 84, "weight": 2490, "acceleration": 15.7}'

26.929889872154185
```

Any questions? [chat with us](https://gitter.im/cortexlabs/cortex).
