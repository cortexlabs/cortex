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
        if config.get("dest_s3_dir"):
            dest_dir = config["dest_s3_dir"]
            bucket, key = dest_dir.replace("s3://", "").split("/", 1)
            pickle.dump(model, open("model.pkl", "wb"))
            s3 = boto3.client("s3")
            s3.upload_file("model.pkl", bucket, os.path.join(key, "model.pkl"))
            print(f"model uploaded to {dest_dir}/model.pkl")
