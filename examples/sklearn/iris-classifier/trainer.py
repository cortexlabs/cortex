# Train the model

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


# Upload the model

import boto3
import pickle

pickle.dump(model, open("model.pkl", "wb"))
s3 = boto3.client("s3")
s3.upload_file("model.pkl", "my-bucket", "model.pkl")
