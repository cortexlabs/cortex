import boto3
import pickle

from sklearn.datasets import load_iris
from sklearn.model_selection import train_test_split
from sklearn.linear_model import LogisticRegression

# Train the model

iris = load_iris()
data, labels = iris.data, iris.target
training_data, test_data, training_labels, test_labels = train_test_split(data, labels)

model = LogisticRegression(solver="lbfgs", multi_class="multinomial", max_iter=1000)
model.fit(training_data, training_labels)
accuracy = model.score(test_data, test_labels)
print("accuracy: {:.2f}".format(accuracy))

# Upload the model

pickle.dump(model, open("model.pkl", "wb"))
s3 = boto3.client("s3")
s3.upload_file("model.pkl", "cortex-examples", "sklearn/iris-classifier/model.pkl")
