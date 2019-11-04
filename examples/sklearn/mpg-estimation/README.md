# Deploy a scikit-learn linear regression model

This example shows how to deploy a sklearn linear regression model trained on the MPG dataset.

## Predictor

We implement Cortex's Predictor interface to load the model and make predictions. Cortex will use this implementation to serve the model as an autoscaling API.

### Initialization

We can place our code to download and initialize the model in the `init()` function:

```python
# predictor.py

model = None

def init(model_path, metadata):
    global model
    model = load(model_path)
```

### Predict

The `predict()` function will be triggered once per request. We extract the features from the sample sent in the request, feed them to the model, and respond with a predicted mpg:

```python
# predictor.py

def predict(sample, metadata):
    input_array = [
        sample["cylinders"],
        sample["displacement"],
        sample["horsepower"],
        sample["weight"],
        sample["acceleration"],
    ]

    result = model.predict([input_array])
    return np.asscalar(result)
```

See [predictor.py](./src/predictor.py) for the complete code.

## Define a deployment

A `deployment` specifies a set of resources that are deployed together. An `api` makes our implementation available as a web service that can serve real-time predictions. This configuration will deploy the implementation specified in `predictor.py`. Note that the `metadata` will be passed into the `init()` function.

```yaml
# cortex.yaml

- kind: deployment
  name: auto

- kind: api
  name: mpg
  predictor:
    path: predictor.py
    metadata:
      model: s3://cortex-examples/sklearn/mpg-estimation/linreg.joblib
  tracker:
    model_type: regression
```

## Deploy to AWS

`cortex deploy` takes the declarative configuration from `cortex.yaml` and creates it on the cluster.

```bash
$ cortex deploy

deployment started
```

Behind the scenes, Cortex containerizes our implementation, makes it servable using Flask, exposes the endpoint with a load balancer, and orchestrates the workload on Kubernetes.

We can track the status of a deployment using `cortex get`:

```bash
$ cortex get mpg --watch

status   up-to-date   available   requested   last update   avg latency
live     1            1           1           9m            -
```

The output above indicates that one replica of the API was requested and is available to serve predictions. Cortex will automatically launch more replicas if the load increases and spin down replicas if there is unused capacity.

## Serve real-time predictions

We can use `curl` to test our prediction service:

```bash
$ cortex get mpg

url: http://***.amazonaws.com/auto/mpg

$ curl http://***.amazonaws.com/auto/mpg \
    -X POST -H "Content-Type: application/json" \
    -d '{"cylinders": 4, "displacement": 135, "horsepower": 84, "weight": 2490, "acceleration": 15.7}'

26.929889872154185
```

Any questions? [chat with us](https://gitter.im/cortexlabs/cortex).
