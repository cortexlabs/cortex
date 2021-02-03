# TaskAPI

Create APIs that can perform arbitrary tasks like training or fine-tuning a model.

## Implement

```python
# train_iris.py

import cortex

def train_iris(config):
    import os

    import boto3, pickle
    from sklearn.datasets import load_iris
    from sklearn.model_selection import train_test_split
    from sklearn.linear_model import LogisticRegression

    s3_filepath = config["dest_s3_dir"]
    bucket, key = s3_filepath.replace("s3://", "").split("/", 1)

    # get the dataset
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
```

```python
# requirements.txt

boto3
scikit-learn==0.23.2
```

```yaml
# train_iris.yaml

- name: train_iris
  kind: TaskAPI
  definition:
    path: train_iris.py
```

## Deploy

```bash
$ python task.py
```

## Describe

```bash
$ cortex get train_iris
```

## Submit a job

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

## Monitor the job

```bash
$ cortex get trainer 69b183ed6bdf3e9b
```

## View the results

Once the job is complete, you should be able to find the trained model in the directory you've specified.

## Delete

```bash
$ cortex delete train_iris
```
