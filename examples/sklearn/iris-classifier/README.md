# Deploy an iris classification scikit-learn model as a web service

This example shows how to deploy a classifier trained on the famous [iris data set](https://archive.ics.uci.edu/ml/datasets/iris) using scikit-learn.

## Train your model

Create a Python file `trainer.py` and use scikit-learn's `LogisticRegression` to train your model:

```python
# trainer.py

from sklearn.datasets import load_iris
from sklearn.model_selection import train_test_split
from sklearn.linear_model import LogisticRegression

iris = load_iris()
data, labels = iris.data, iris.target
training_data, test_data, training_labels, test_labels = train_test_split(data, labels)

model = LogisticRegression(solver="lbfgs", multi_class="multinomial")
model.fit(training_data, training_labels)
accuracy = model.score(test_data, test_labels)
print("accuracy: {:.2f}".format(accuracy))
```

Add code to pickle (you can use other serialization libraries such as joblib) and upload your model to S3. boto3 will need access to valid AWS credentials. You should also change "cortex-examples" to your bucket's name:

```python
# trainer.py

import boto3
import pickle

pickle.dump(model, open("model.pkl", "wb"))
s3 = boto3.client("s3")
s3.upload_file("model.pkl", "cortex-examples", "model.pkl")
```

Run the script locally:

```bash
# Install scikit-learn
$ pip3 install sklearn

# Run the script
$ python3 trainer.py
```

## Define a predictor

Cortex's Predictor Interface allows Cortex to serve models from any framework by implementing them in a Python file. To use the Predictor Interface, create another Python file `predictor.py` and add code to download your pickled model from S3:

```python
# predictor.py

import boto3
import pickle

object = boto3.client("s3").get_object(
    Bucket="cortex-examples", Key="sklearn/iris-classifier/model.pkl"
)
model = pickle.loads(object["Body"].read())
```

Cortex's Predictor Interface will search within your `predictor.py` file for a `predict()` function that accepts inputs and serves predictions. Add a prediction function that will accept a JSON sample and return a prediction from your model:

```python
# predictor.py

import numpy

labels = ["iris-setosa", "iris-versicolor", "iris-virginica"]

def predict(sample, metadata):
    input_array = numpy.array(
        [
            sample["sepal_length"],
            sample["sepal_width"],
            sample["petal_length"],
            sample["petal_width"],
        ]
    )

    label_id = model.predict([input_array])[0]
    return labels[label_id]
```

## Specify Python dependencies

Create a `requirements.txt` file to specify the dependencies needed by `predictor.py`. Cortex will automatically install them into your runtime once you deploy:

```text
boto3
numpy
```

You can skip dependencies that are [pre-installed](https://www.cortex.dev/deployments/predictor#pre-installed-packages) to speed up the deployment process. Note that `pickle` is part of the Python standard library so it doesn't need to be included.

## Define a deployment

A `deployment` specifies a set of resources that are deployed together. An `api` makes our implementation available as a web service that can serve real-time predictions. Cortex will configure your `deployment` and your `api` according to the instructions passed to it via a `cortex.yaml` configuration file. Create the following `cortex.yaml` to deploy the implementation specified in `predictor.py`:

```yaml
# cortex.yaml

- kind: deployment
  name: iris

- kind: api
  name: classifier
  predictor:
    path: predictor.py
```

## Deploy to AWS

`cortex deploy` takes the declarative configuration from `cortex.yaml` and creates it on the cluster:

```bash
$ cortex deploy

creating classifier api
```

Behind the scenes, Cortex containerizes your implementation, makes it servable using Flask, exposes the endpoint with a load balancer, and orchestrates the workload on Kubernetes.

Track the status of your deployment using `cortex get`:

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

## Add prediction tracking

Add a `tracker` to your `cortex.yaml` and specify that this is a classification model:

```yaml
# cortex.yaml

- kind: deployment
  name: iris

- kind: api
  name: classifier
  predictor:
    path: predictor.py
  tracker:
    model_type: classification
```

Run `cortex deploy` again to update your API with this configuration:

```bash
$ cortex deploy

updating classifier api


$ cortex get classifier --watch

status     up-to-date   available   requested   last update   avg latency
updating   0            1           1           3s            -
```

After making more predictions, your `cortex get` command will show information about your API's past predictions:

```bash
$ cortex get classifier --watch

status   up-to-date   available   requested   last update   avg latency
live     1            1           1           16s           123ms

class     count
positive  8
negative  4
```

## Add compute resources

This model is fairly small but larger models may require more compute resources. You can configure this in your `cortex.yaml`:

```yaml
- kind: deployment
  name: iris

- kind: api
  name: classifier
  predictor:
    path: predictor.py
  tracker:
    model_type: classification
  compute:
    cpu: 1
    mem: 1G
```

You could also configure GPU compute here if your cluster supports it. Adding compute resources may help reduce your inference latency. Run `cortex deploy` again to update your API with this configuration:

```bash
$ cortex deploy

updating classifier api


$ cortex get classifier --watch

status     up-to-date   available   requested   last update   avg latency
updating   0            1           1           3s            88ms

class     count
positive  8
negative  4
```

## Clean up

Run `cortex delete` to spin down your API:

```bash
$ cortex delete

deleting classifier api
```

Running `cortex delete` will free up cluster resources and allow Cortex to scale to the minimum number of instances you specified during cluster installation. It will not spin down your cluster.

Any questions? [chat with us](https://gitter.im/cortexlabs/cortex).
