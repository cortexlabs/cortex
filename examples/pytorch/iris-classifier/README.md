# Deploy a PyTorch iris classifier

This example shows how to deploy a classifier trained on the famous [iris data set](https://archive.ics.uci.edu/ml/datasets/iris) in PyTorch. The PyTorch model being deployed can be found [here](./src/my_model.py).

## Predictor

We implement Cortex's Predictor interface to load the model and make predictions. Cortex will use this implementation to serve the model as an autoscaling API.

### Initialization

We can place our code to download and initialize the model in the `init()` function. The PyTorch model class is defined in [src/model.py](./src/model.py), and we assume that we've already trained the model and uploaded the state_dict (weights) to S3.

```python
# predictor.py

from model import IrisNet

# instantiate the model
model = IrisNet()

# define the labels
labels = ["iris-setosa", "iris-versicolor", "iris-virginica"]

def init(model_path, metadata):
    # model_path is a local path pointing to your model weights file
    model.load_state_dict(torch.load(model_path))
    model.eval()
```

### Predict

The `predict()` function will be triggered once per request. We extract the features from the sample sent in the request, feed them to the model, and respond with a human-readable label:

```python
# predictor.py

def predict(sample, metadata):
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

See [predictor.py](./src/predictor.py) for the complete code.

## Define a deployment

A `deployment` specifies a set of resources that are deployed together. An `api` makes our implementation available as a web service that can serve real-time predictions. This configuration will deploy the implementation specified in `predictor.py`. Note that the `metadata` will be passed into the `init()` function, and we specify the `python_path` in the so  we can import our model as `from my_model import IrisNet` instead of `from src.my_model import IrisNet` in `predictor.py`.

```yaml
# cortex.yaml

- kind: deployment
  name: iris

- kind: api
  name: classifier
  predictor:
    path: src/predictor.py
    python_path: src/
    model: s3://cortex-examples/pytorch/iris-classifier/weights.pth
  tracker:
    model_type: classification
```

## Deploy to AWS

`cortex deploy` takes the declarative configuration from `cortex.yaml` and creates it on the cluster:

```bash
$ cortex deploy

deployment started
```

Behind the scenes, Cortex containerizes our implementation, makes it servable using Flask, exposes the endpoint with a load balancer, and orchestrates the workload on Kubernetes.

We can track the status of a deployment using `cortex get`:

```bash
$ cortex get classifier --watch

status   up-to-date   available   requested   last update   avg latency
live     1            1           1           8s            -
```

The output above indicates that one replica of the API was requested and is available to serve predictions. Cortex will automatically launch more replicas if the load increases and spin down replicas if there is unused capacity.

## Serve real-time predictions

We can use `curl` to test our prediction service:

```bash
$ cortex get classifier

endpoint: http://***.amazonaws.com/iris/classifier

$ curl http://***.amazonaws.com/iris/classifier \
    -X POST -H "Content-Type: application/json" \
    -d '{"sepal_length": 5.2, "sepal_width": 3.6, "petal_length": 1.4, "petal_width": 0.3}'

"iris-setosa"
```

Any questions? [chat with us](https://gitter.im/cortexlabs/cortex).
