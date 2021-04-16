# TaskAPI

Create APIs that can perform arbitrary tasks like training or fine-tuning a model.

## Implement

```python
# train_iris.py

import os
import boto3
import pickle
from sklearn.datasets import load_iris
from sklearn.model_selection import train_test_split
from sklearn.linear_model import LogisticRegression


class Task:
    def __call__(self, config):
        # get the iris flower dataset
        iris = load_iris()
        data, labels = iris.data, iris.target
        training_data, test_data, training_labels, test_labels = train_test_split(data, labels)
        print("loaded dataset")

        # train the model
        model = LogisticRegression(solver="lbfgs", multi_class="multinomial", max_iter=1000)
        model.fit(training_data, training_labels)
        accuracy = model.score(test_data, test_labels)
        print("model trained; accuracy: {:.2f}".format(accuracy))

        # upload the model
        dest_dir = config["dest_s3_dir"]
        bucket, key = dest_dir.replace("s3://", "").split("/", 1)
        pickle.dump(model, open("model.pkl", "wb"))
        s3 = boto3.client("s3")
        s3.upload_file("model.pkl", bucket, os.path.join(key, "model.pkl"))
        print(f"model uploaded to {dest_dir}/model.pkl")
```

```python
# requirements.txt

boto3
scikit-learn==0.23.2
```

```yaml
# cortex.yaml

- name: train-iris
  kind: TaskAPI
  definition:
    path: train_iris.py
```

## Deploy

```bash
cortex deploy
```

## Describe

```bash
cortex get train-iris

# > endpoint: http://***.elb.us-west-2.amazonaws.com/train-iris
```

## Submit a job

You can submit a job by making a POST request to the Task API's endpoint.

Using `curl`:

```bash
export TASK_API_ENDPOINT=<TASK_API_ENDPOINT>  # e.g. export TASK_API_ENDPOINT=https://***.elb.us-west-2.amazonaws.com/train-iris
export DEST_S3_DIR=<YOUR_S3_DIRECTORY>  # e.g. export DEST_S3_DIR=s3://my-bucket/dir

curl $TASK_API_ENDPOINT \
    -X POST -H "Content-Type: application/json" \
    -d "{\"config\": {\"dest_s3_dir\": \"$DEST_S3_DIR\"}}"
# > {"job_id":"69b183ed6bdf3e9b","api_name":"train-iris",...}
```

Or, using Python `requests`:

```python
import cortex
import requests

cx = cortex.client("aws")  # "aws" is the name of the Cortex environment used in this example
task_endpoint = cx.get_api("train-iris")["endpoint"]

dest_s3_dir =  # S3 directory where the model will be uploaded, e.g. "s3://my-bucket/dir"
job_spec = {
    "config": {
        "dest_s3_dir": dest_s3_dir
    }
}
response = requests.post(task_endpoint, json=job_spec)
print(response.text)
# > {"job_id":"69b183ed6bdf3e9b","api_name":"train-iris",...}
```

## Monitor the job

```bash
cortex get train-iris 69b183ed6bdf3e9b
```

## View the results

Once the job is complete, you should be able to find the trained model in the directory you've specified.

## Delete

```bash
cortex delete train-iris
```
