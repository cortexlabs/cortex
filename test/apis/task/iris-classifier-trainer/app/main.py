import json
import pickle
import re
import os
import boto3

from sklearn.datasets import load_iris
from sklearn.model_selection import train_test_split
from sklearn.linear_model import LogisticRegression


def main():
    with open("/cortex/spec/job.json", "r") as f:
        job_spec = json.load(f)
    print(json.dumps(job_spec, indent=2))

    # get metadata
    config = job_spec["config"]
    job_id = job_spec["job_id"]
    s3_path = None
    if config is not None and "dest_s3_dir" in config:
        s3_path = config["dest_s3_dir"]

    # Train the model
    iris = load_iris()
    data, labels = iris.data, iris.target
    training_data, test_data, training_labels, test_labels = train_test_split(data, labels)

    model = LogisticRegression(solver="lbfgs", multi_class="multinomial", max_iter=1000)
    model.fit(training_data, training_labels)
    accuracy = model.score(test_data, test_labels)
    print("accuracy: {:.2f}".format(accuracy))

    # Upload the model
    if s3_path:
        pickle.dump(model, open("model.pkl", "wb"))
        bucket, key = re.match("s3://(.+?)/(.+)", s3_path).groups()
        s3 = boto3.client("s3")
        s3.upload_file("model.pkl", bucket, os.path.join(key, job_id, "model.pkl"))
    else:
        print("not uploading the model to the s3 bucket")


if __name__ == "__main__":
    main()
