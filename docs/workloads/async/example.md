# AsyncAPI

Create APIs that process your workloads asynchronously.

## Implementation

Create a folder for your API. In this case, we are deploying an iris-classifier AsyncAPI. This folder will have the
following structure:

```shell
./iris-classifier
├── cortex.yaml
├── predictor.py
└── requirements.txt
```

We will now create the necessary files:

```bash
mkdir iris-classifier && cd iris-classifier
touch predictor.py requirements.txt cortex.yaml
```

```python
# predictor.py

import os
import pickle
from typing import Dict, Any

import boto3
from botocore import UNSIGNED
from botocore.client import Config

labels = ["setosa", "versicolor", "virginica"]


class PythonPredictor:
    def __init__(self, config):
        if os.environ.get("AWS_ACCESS_KEY_ID"):
            s3 = boto3.client("s3")  # client will use your credentials if available
        else:
            s3 = boto3.client("s3", config=Config(signature_version=UNSIGNED))  # anonymous client

        s3.download_file(config["bucket"], config["key"], "/tmp/model.pkl")
        self.model = pickle.load(open("/tmp/model.pkl", "rb"))

    def predict(self, payload: Dict[str, Any]) -> Dict[str, str]:
        measurements = [
            payload["sepal_length"],
            payload["sepal_width"],
            payload["petal_length"],
            payload["petal_width"],
        ]

        label_id = self.model.predict([measurements])[0]

        # result must be json serializable
        return {"label": labels[label_id]}
```

```
# requirements.txt

boto3
```

```yaml
# text_generator.yaml

- name: iris-classifier
  kind: AsyncAPI
  predictor:
    type: python
    path: predictor.py
```

## Deploy

We can now deploy our API with the `cortex deploy` command. This command can be re-run to update your API configuration
or predictor implementation.

```bash
cortex deploy cortex.yaml

# creating iris-classifier (AsyncAPI)
#
# cortex get                  (show api statuses)
# cortex get iris-classifier  (show api info)
```

## Monitor

To check whether the deployed API is ready, we can run the `cortex get` command with the `--watch` flag.

```bash
cortex get iris-classifier --watch

# status     up-to-date   requested   last update
# live       1            1           10s
#
# endpoint: http://<load_balancer_url>/iris-classifier
#
# api id                                                         last deployed
# 6992e7e8f84469c5-d5w1gbvrm5-25a7c15c950439c0bb32eebb7dc84125   10s
```

## Submit a workload

Now we want to submit a workload to our deployed API. We will start by creating a file with a JSON request payload, in
the format expected by our `iris-classifier` predictor implementation.

This is the JSON file we will submit to our iris-classifier API.

```bash
# sample.json
{
    "sepal_length": 5.2,
    "sepal_width": 3.6,
    "petal_length": 1.5,
    "petal_width": 0.3
}
```

Once we have our sample request payload, we will submit it with a `POST` request to the endpoint URL previously
displayed in the `cortex get` command. We will quickly get a request `id` back.

```bash
curl -X POST http://<load_balancer_url>/iris-classifier -H "Content-Type: application/json" -d '@./sample.json'

# {"id": "659938d2-2ef6-41f4-8983-4e0b7562a986"}
```

## Retrieve the result

The obtained request id will allow us to check the status of the running payload and retrieve its result. To do so, we
submit a `GET` request to the same endpoint URL with an appended `/<id>`.

```bash
curl http://<load_balancer_url>/iris-classifier/<id>  # <id> is the request id that was returned in the previous POST request

# {
#   "id": "659938d2-2ef6-41f4-8983-4e0b7562a986",
#   "status": "completed",
#   "result": {"label": "setosa"},
#   "timestamp": "2021-03-16T15:50:50+00:00"
# }
```

Depending on the status of your workload, you will get different responses back. The possible workload status
are `in_queue | in_progress | failed | completed`. The `result` and `timestamp` keys are returned if the status
is `completed`.

It is also possible to setup a webhook in your predictor to get the response sent to a pre-defined web server once the
workload completes or fails. You can read more about it in the [webhook documentation](./webhooks.md).

## Stream logs

If necessary, you can stream the logs from a random running pod from your API with the `cortex logs` command. This is
intended for debugging purposes only. For production logs, you can view the logs in the logging solution of the cloud
provider your cluster is deployed in.

```bash
cortex logs iris-classifier
```

## Delete the API

Finally, you can delete your API with a simple `cortex delete` command.

```bash
cortex delete iris-classifier
```
