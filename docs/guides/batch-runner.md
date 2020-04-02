# Add a batch runner API

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

We have plans to support a batch interface to Cortex APIs ([#523](https://github.com/cortexlabs/cortex/issues/523)), but until that's implemented, it is possible to implement a batch runner which receives the batch request, splits it into individual requests, and sends them to the prediction API.

_Note: this is experimental. Also, this behavior can be implemented outside of Cortex, e.g. in your backend server if you have one._

This example assumes you have deployed an iris-classifier API, e.g. [examples/sklearn/iris-classifier](https://github.com/cortexlabs/cortex/tree/master/examples/sklearn/iris-classifier) or [examples/tensorflow/iris-classifier](https://github.com/cortexlabs/cortex/tree/master/examples/tensorflow/iris-classifier).

Create a new directory (outside of the iris example directory) with the files listed below, and run `cortex deploy` in that directory to deploy the batch API. Run `python test.py http://***.us-west-2.elb.amazonaws.com/iris-classifier-batch` to submit a batch of requests to the batch api (replace `***` with your actual endpoint). You can still send individual requests to the prediction API (bypassing the batch API) if you'd like.

Feel free to reach out on [gitter](https://gitter.im/cortexlabs/cortex) if you have questions.

## `batch.py`

```python
import requests
import time
import json
from concurrent.futures import ThreadPoolExecutor


class PythonPredictor:
    def __init__(self, config):
        self.endpoint = config["endpoint"]  # endpoint is passed in from the API configuration
        print(f"endpoint: {self.endpoint}")

    def predict(self, payload):
        # For this example, payload will be the full list of iris-classifier prediction requests.
        # We will send each as it's own prediction request to the API, however it may be beneficial
        # to create batches, e.g. a single request to the classifier may contain 100 individual samples.
        # For example, payload could be a list of image URLs, e.g. ["url1", "url2", "url3", "url4", "url5", "url6"]
        # then batches could be a list where each item is a list of the image URLs to make in a single inference request
        # e.g. [["url1", "url2", "url3"], ["url4", "url5", "url6"]] would correspond to two requests to your inference API, each containing three URLs
        batches = payload

        # Increasing max_workers will increase how many replicas will be used for the batch request.
        # Assuming default values for target_replica_concurrency, workers_per_replica, and threads_per_worker
        # in your prediction API, the number of replicas created will be equal to max_workers.
        # If you have changed these values, the number of replicas created will be equal to max_workers / target_replica_concurrency
        # (note that the default value of target_replica_concurrency is workers_per_replica * threads_per_worker).
        # If max_workers starts to get large, you will also want to set the inference API's max_replica_concurrency to avoid long and imbalanced queue lengths
        # Here are the autoscaling docs: https://www.cortex.dev/deployments/autoscaling
        with ThreadPoolExecutor(max_workers=5) as executor:
            results = executor.map(self.make_request, batches)

        # This is to wait for all the results to be completed
        list(results)

    # make_request() receives a single element from the batches list.
    # In this case it's just a single sample, but in general it should be the info necessary
    # to make one request to the inference API, e.g. a list of URLs
    def make_request(self, batch):
        print("making a request")
        num_attempts = 0

        while True:
            num_attempts += 1

            # Make the actual inference request
            response = requests.post(self.endpoint, data=json.dumps(batch))

            if response.status_code == 200:
                break

            print(f"got response code {response.status_code}, retrying...")

            # Enforce a maximum number of retries / timeout here so this request can't go on forever.
            # The total timeout should be at least e.g. 10 minutes if you want to allow for new instances to spin up
            if num_attempts >= 60:
                # you may want to handle this case differently
                print("max attempts exceeded, giving up")
                return

            time.sleep(10)

        # This is your inference API response
        print(f"done with request: {response.text}")
```

## `cortex.yaml`

```yaml
- name: iris-classifier-batch
  predictor:
    type: python
    path: batch.py
    config:  # you can pass in your API endpoint like this (replace ***):
      endpoint: http://***.us-west-2.elb.amazonaws.com/iris-classifier
  autoscaling:
    max_replicas: 1  # this API may need to autoscale depending on how many batch requests, but disable it to start
    threads_per_worker: 1  # set this to the number of batch requests you'd like to be able to be able to work on at a time
                           # once that number is exceeded, they will be queued, which may be ok
                           # setting this too high may lead to out of memory errors
  compute:
    cpu: 500m  # reserve some CPU (you may be able to decrease this, or you may have to increase it)
    mem: 1Gi   # reserve some memory (you may be able to decrease this, or you may have to increase it)
```

## `test.py`

```python
import sys
import requests
import json

if len(sys.argv) != 2:
    print("usage: python test.py BATCH_API_URL")
    print("e.g. python test.py http://***.us-west-2.elb.amazonaws.com/iris-classifier-batch")
    sys.exit(1)

batch_endpoint = sys.argv[1]

# The full list of requests to make
data = [
    {"sepal_length": 5.2, "sepal_width": 3.6, "petal_length": 1.5, "petal_width": 0.3},
    {"sepal_length": 7.1, "sepal_width": 3.3, "petal_length": 4.8, "petal_width": 1.5},
    {"sepal_length": 6.4, "sepal_width": 3.4, "petal_length": 6.1, "petal_width": 2.6},
]

# This timeout and try/catch is so that you can send the request and not wait for it, since
# cortex doesn't support asynchronous requests yet
try:
    response = requests.post(batch_endpoint, data=json.dumps(data), timeout=0.05)
    if response.status_code != 200:
        print("an error occurred:")
        print(response.text)
except requests.exceptions.ReadTimeout:
    pass
```
