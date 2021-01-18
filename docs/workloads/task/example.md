# TaskAPI

Deploy a task API that trains a model on the iris flower dataset and uploads it to an S3 bucket.

## Key features

* Lambda-style execution
* Task monitoring
* Scale to 0

## How it works

### Install cortex

```bash
$ pip install cortex
```

### Spin up a cluster on AWS

```bash
$ cortex cluster up
```

### Define a task API

```python
# task.py

import cortex

def train_iris_model(config):
    import os

    import boto3, pickle
    from sklearn.datasets import load_iris
    from sklearn.model_selection import train_test_split
    from sklearn.linear_model import LogisticRegression

    s3_filepath = config["dest_s3_dir"]
    bucket, key = s3_filepath.replace("s3://", "").split("/", 1)

    # get iris flower dataset
    iris = load_iris()
    data, labels = iris.data, iris.target
    training_data, test_data, training_labels, test_labels = train_test_split(data, labels)

    # train the model
    model = LogisticRegression(solver="lbfgs", multi_class="multinomial")
    model.fit(training_data, training_labels)
    accuracy = model.score(test_data, test_labels)
    print("accuracy: {:.2f}".format(accuracy))

    # upload the model
    pickle.dump(model, open("model.pkl", "wb"))
    s3 = boto3.client("s3")
    s3.upload_file("model.pkl", bucket, os.path.join(key, "model.pkl"))

requirements = ["scikit-learn==0.23.2", "boto3"]

api_spec = {
    "name": "trainer",
    "kind": "TaskAPI",
}

cx = cortex.client("aws")
cx.create_api(api_spec, task=train_iris_model, requirements=requirements)
```

### Deploy to your Cortex cluster on AWS

```bash
$ python task.py
```

### Describe the task API

```bash
$ cortex get trainer
```

### Submit a job

```python
import cortex
import requests

cx = cortex.client("aws")
task_endpoint = cx.get_api("trainer")["endpoint"]

dest_s3_dir = # specify S3 directory where the trained model will get pushed to
job_spec = {
    "config": {
        "dest_s3_dir": dest_s3_dir
    }
}
response = requests.post(task_endpoint, json=job_spec)

print(response.text)
# > {"job_id":"69b183ed6bdf3e9b","api_name":"trainer", "config": {"dest_s3_dir": ...}}
```

### Monitor the job

```bash
$ cortex get trainer 69b183ed6bdf3e9b
```

### View the results

Once the job is complete, you should be able to find the trained model of the task job in the S3 directory you've specified.

### Delete the Task API

```bash
$ cortex delete trainer
```
