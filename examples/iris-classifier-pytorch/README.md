# Deploy an iris classifier in PyTorch

This example shows how to deploy a classifier trained on the famous [iris data set](https://archive.ics.uci.edu/ml/datasets/iris) in PyTorch. The PyTorch model being deployed can be found [here](./src/model.py).


## Inference

We implement Cortex's python inference interface that describes how to load the model and make predictions using the model. Cortex will use this implementation to serve your model as an API of autoscaling replicas. We specify a `requirements.txt` to install dependencies necessary to implement the Cortex inference interface.

### Initialization

Cortex executes the python implementation once per replica startup. We can place our initializations in the body of the implementation. The PyTorch model class is defined in [src/my_model.py](./src/my_model.py). Let us instantiate our model and define the labels.
```
from my_model import IrisNet

model = IrisNet()

labels = ["iris-setosa", "iris-versicolor", "iris-virginica"]
```

Let's assume that we have already trained a model uploaded the state_dict (weights) to S3. We get the location of the weights from the metadata in the configuration and initialize our model with weights from S3 in the `init` function. Cortex calls the `init` once per replica startup.
```
def init(metadata):
    s3 = boto3.client("s3")
    s3.download_file(metadata["bucket"], metadata["key"], "iris_model.pth")
    model.load_state_dict(torch.load("iris_model.pth"))
    model.eval()
```

### Predict

The `predict` function will be triggered once per request to run the Iris model on the request payload and respond with a prediction. In the `predict` function, we extract the iris features from the sample sent in the request and respond with a human readable label.

```python
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

See `inference.py` for the complete code.

## Define a deployment

A `deployment` specifies a set of resources that are deployed as a single unit. An `api` makes the Cortex python implementation available as a web service that can serve real-time predictions. The metadata specified in this configuration will passed into the `init` function in `inference.py` for model initialization. We specify the `python_root` in the deployment so that we can import our model as `from my_model import IrisNet` instead of `from src.my_model import IristNet` in `inference.py`.

```yaml
- kind: deployment
  name: iris
  python_root: src/

- kind: api
  name: classifier
  python:
    inference: src/inference.py
  metadata:
    bucket: data-vishal
    key: iris_model.pth
```

## Deploy to AWS

`cortex deploy` takes the declarative configuration from `cortex.yaml` and creates it on the cluster:

```bash
$ cortex deploy

deployment started
```

Behind the scenes, Cortex containerizes the python inference implementation, makes it servable using Flask, exposes the endpoint with a load balancer, and orchestrates the workload on Kubernetes.

You can track the status of a deployment using `cortex get`:

```bash
$ cortex get classifier --watch

status   up-to-date   available   requested   last update   avg latency
live     1            1           1           8s            -
```

The output above indicates that one replica of the API was requested and one replica is available to serve predictions. Cortex will automatically launch more replicas if the load increases and spin down replicas if there is unused capacity.

## Serve real-time predictions

```bash
$ cortex get classifier

url: http://***.amazonaws.com/iris/classifier

$ curl http://***.amazonaws.com/iris/classifier \
    -X POST -H "Content-Type: application/json" \
    -d '{"sepal_length": 5.2, "sepal_width": 3.6, "petal_length": 1.4, "petal_width": 0.3}'

"iris-setosa"
```

Any questions? [chat with us](https://gitter.im/cortexlabs/cortex).
