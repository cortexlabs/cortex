import os

import boto3, pickle
from sklearn.datasets import load_iris
from sklearn.model_selection import train_test_split
from sklearn.linear_model import LogisticRegression


class Task:
    def __call__(self, config):
        # get iris flower dataset
        iris = load_iris()
        data, labels = iris.data, iris.target
        training_data, test_data, training_labels, test_labels = train_test_split(data, labels)

        # train the model
        model = LogisticRegression(solver="lbfgs", multi_class="multinomial")
        model.fit(training_data, training_labels)
        accuracy = model.score(test_data, test_labels)
        print("accuracy: {:.2f}".format(accuracy))

        # upload the model if so specified
        if config.get("upload_model"):
            s3_filepath = config["dest_s3_dir"]
            bucket, key = s3_filepath.replace("s3://", "").split("/", 1)
            pickle.dump(model, open("model.pkl", "wb"))
            s3 = boto3.client("s3")
            s3.upload_file("model.pkl", bucket, os.path.join(key, "model.pkl"))
