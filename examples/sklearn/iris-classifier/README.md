# Deploy a scikit-learn model as a web service

_WARNING: you are on the master branch, please refer to the examples on the branch that matches your `cortex version`_

This example shows how to deploy a classifier trained on the famous [iris data set](https://archive.ics.uci.edu/ml/datasets/iris) using scikit-learn.

<br>

## Train your model

1. Create a Python file `trainer.py`.
2. Use scikit-learn's `LogisticRegression` to train your model.
3. Add code to pickle your model (you can use other serialization libraries such as joblib).
4. Upload it to S3 (boto3 will need access to valid AWS credentials).

```python
import boto3
import pickle
from sklearn.datasets import load_iris
from sklearn.model_selection import train_test_split
from sklearn.linear_model import LogisticRegression

# Train the model

iris = load_iris()
data, labels = iris.data, iris.target
training_data, test_data, training_labels, test_labels = train_test_split(data, labels)

model = LogisticRegression(solver="lbfgs", multi_class="multinomial")
model.fit(training_data, training_labels)
accuracy = model.score(test_data, test_labels)
print("accuracy: {:.2f}".format(accuracy))

# Upload the model

pickle.dump(model, open("model.pkl", "wb"))
s3 = boto3.client("s3")
s3.upload_file("model.pkl", "my-bucket", "sklearn/iris-classifier/model.pkl")
```

Run the script locally:

```bash
# Install scikit-learn and boto3
$ pip3 install sklearn boto3

# Run the script
$ python3 trainer.py
```

<br>

## Implement a predictor

1. Create another Python file `predictor.py`.
2. Define a Predictor class with a constructor that loads and initializes your pickled model.
3. Add a predict function that will accept a payload and return a prediction from your model.

```python
# predictor.py

import boto3
import pickle
import re

labels = ["setosa", "versicolor", "virginica"]


class PythonPredictor:
    def __init__(self, config):
        bucket, key = re.match("s3://(.+?)/(.+)", config["model"]).groups()
        s3 = boto3.client("s3")
        s3.download_file(bucket, key, "model.pkl")

        self.model = pickle.load(open("model.pkl", "rb"))

    def predict(self, payload):
        measurements = [
            payload["sepal_length"],
            payload["sepal_width"],
            payload["petal_length"],
            payload["petal_width"],
        ]

        label_id = self.model.predict([measurements])[0]
        return labels[label_id]
```

<br>

## Specify Python dependencies

Create a `requirements.txt` file to specify the dependencies needed by `predictor.py`. Cortex will automatically install them into your runtime once you deploy:

```python
# requirements.txt

boto3
```

You can skip dependencies that are [pre-installed](../../../docs/deployments/python.md#pre-installed-packages) to speed up the deployment process. Note that `pickle` is part of the Python standard library so it doesn't need to be included.

<br>

## Configure a deployment

Create a `cortex.yaml` file and add the configuration below. A `deployment` specifies a set of resources that are deployed together. An `api` provides a runtime for inference and makes our `predictor.py` implementation available as a web service that can serve real-time predictions:

```yaml
# cortex.yaml

- kind: deployment
  name: iris

- kind: api
  name: classifier
  python:
    predictor: predictor.py
    config:
      model: s3://cortex-examples/sklearn/iris-classifier/model.pkl
```

<br>

## Deploy to AWS

`cortex deploy` takes the declarative configuration from `cortex.yaml` and creates it on your Cortex cluster:

```bash
$ cortex deploy

creating classifier api
```

Track the status of your deployment using `cortex get`:

```bash
$ cortex get classifier --watch

status   up-to-date   requested   last update   avg latency
live     1            1           8s            -

endpoint: http://***.amazonaws.com/iris/classifier
```

The output above indicates that one replica of the API was requested and is available to serve predictions. Cortex will automatically launch more replicas if the load increases and spin down replicas if there is unused capacity.

You can also stream logs from your API:

```bash
$ cortex logs classifier
```

<br>

## Serve real-time predictions

We can use `curl` to test our prediction service:

```bash
$ curl http://***.amazonaws.com/iris/classifier \
    -X POST -H "Content-Type: application/json" \
    -d '{"sepal_length": 5.2, "sepal_width": 3.6, "petal_length": 1.4, "petal_width": 0.3}'

"setosa"
```

<br>

## Configure prediction tracking

Add a `tracker` to your `cortex.yaml` and specify that this is a classification model:

```yaml
# cortex.yaml

- kind: deployment
  name: iris

- kind: api
  name: classifier
  python:
    predictor: predictor.py
    config:
      model: s3://cortex-examples/sklearn/iris-classifier/model.pkl
  tracker:
    model_type: classification
```

Run `cortex deploy` again to perform a rolling update to your API with the new configuration:

```bash
$ cortex deploy

updating classifier api
```

After making more predictions, your `cortex get` command will show information about your API's past predictions:

```bash
$ cortex get classifier --watch

status   up-to-date   requested   last update   avg latency
live     1            1           16s           28ms

class        count
setosa       8
versicolor   2
virginica    4
```

<br>

## Configure compute resources

This model is fairly small but larger models may require more compute resources. You can configure this in your `cortex.yaml`:

```yaml
- kind: deployment
  name: iris

- kind: api
  name: classifier
  python:
    predictor: predictor.py
    config:
      model: s3://cortex-examples/sklearn/iris-classifier/model.pkl
  tracker:
    model_type: classification
  compute:
    cpu: 0.5
    mem: 1G
```

You could also configure GPU compute here if your cluster supports it. Adding compute resources may help reduce your inference latency. Run `cortex deploy` again to update your API with this configuration:

```bash
$ cortex deploy

updating classifier api
```

Run `cortex get` again:

```bash
$ cortex get classifier --watch

status   up-to-date   requested   last update   avg latency
live     1            1           16s           24 ms

class        count
setosa       8
versicolor   2
virginica    4
```

<br>

## Add another API

If you trained another model and want to A/B test it with your previous model, simply add another `api` to your configuration and specify the new model:

```yaml
- kind: deployment
  name: iris

- kind: api
  name: classifier
  python:
    predictor: predictor.py
    config:
      model: s3://cortex-examples/sklearn/iris-classifier/model.pkl
  tracker:
    model_type: classification
  compute:
    cpu: 0.5
    mem: 1G

- kind: api
  name: another-classifier
  python:
    predictor: predictor.py
    config:
      model: s3://cortex-examples/sklearn/iris-classifier/another-model.pkl
  tracker:
    model_type: classification
  compute:
    cpu: 0.5
    mem: 1G
```

Run `cortex deploy` to create the new API:

```bash
$ cortex deploy

creating another-classifier api
```

`cortex deploy` is declarative so the `classifier` API is unchanged while `another-classifier` is created:

```bash
$ cortex get --watch

api                  status   up-to-date   requested   last update
classifier           live     1            1           5m
another-classifier   live     1            1           8s
```

## Add a batch API

First, implement `batch-predictor.py` with a `predict` function that can process an array of payloads:

```python
# batch-predictor.py

import boto3
import pickle
import re

labels = ["setosa", "versicolor", "virginica"]


class PythonPredictor:
    def __init__(self, config):
        bucket, key = re.match("s3://(.+?)/(.+)", config["model"]).groups()
        s3 = boto3.client("s3")
        s3.download_file(bucket, key, "model.pkl")

        self.model = pickle.load(open("model.pkl", "rb"))

    def predict(self, payload):
        measurements = [
            [
                sample["sepal_length"],
                sample["sepal_width"],
                sample["petal_length"],
                sample["petal_width"],
            ]
            for sample in payload
        ]

        label_ids = self.model.predict(measurements)
        return [labels[label_id] for label_id in label_ids]
```

Next, add the `api` to `cortex.yaml`:

```yaml
- kind: deployment
  name: iris

- kind: api
  name: classifier
  python:
    predictor: predictor.py
    config:
      model: s3://cortex-examples/sklearn/iris-classifier/model.pkl
  tracker:
    model_type: classification
  compute:
    cpu: 0.5
    mem: 1G

- kind: api
  name: another-classifier
  python:
    predictor: predictor.py
    config:
      model: s3://cortex-examples/sklearn/iris-classifier/another-model.pkl
  tracker:
    model_type: classification
  compute:
    cpu: 0.5
    mem: 1G


- kind: api
  name: batch-classifier
  python:
    predictor: batch-predictor.py
    config:
      model: s3://cortex-examples/sklearn/iris-classifier/model.pkl
  compute:
    cpu: 0.5
    mem: 1G
```

Run `cortex deploy` to create the batch API:

```bash
$ cortex deploy

creating batch-classifier api
```

`cortex get` should show all three APIs now:

```bash
$ cortex get --watch

api                  status   up-to-date   requested   last update
classifier           live     1            1           10m
another-classifier   live     1            1           5m
batch-classifier     live     1            1           8s
```

<br>

## Try a batch prediction

```bash
$ curl http://***.amazonaws.com/iris/classifier \
    -X POST -H "Content-Type: application/json" \
    -d '[
          {
        		"sepal_length": 5.2,
        		"sepal_width": 3.6,
        		"petal_length": 1.5,
        		"petal_width": 0.3
        	},
        	{
        		"sepal_length": 7.1,
        		"sepal_width": 3.3,
        		"petal_length": 4.8,
        		"petal_width": 1.5
        	},
        	{
        		"sepal_length": 6.4,
        		"sepal_width": 3.4,
        		"petal_length": 6.1,
        		"petal_width": 2.6
        	}
        ]'

["setosa","versicolor","virginica"]
```

<br>

## Clean up

Run `cortex delete` to spin down your API:

```bash
$ cortex delete

deleting classifier api
deleting another-classifier api
deleting batch-classifier api
```

Running `cortex delete` will free up cluster resources and allow Cortex to scale down to the minimum number of instances you specified during cluster installation. It will not spin down your cluster.

Any questions? [chat with us](https://gitter.im/cortexlabs/cortex).
