"""
Requirements.txt

onnxmltools
pandas
scikit-learn
skl2onnx
"""
import numpy as np
from sklearn.datasets import load_iris
from sklearn.model_selection import train_test_split
from onnxconverter_common.data_types import FloatTensorType
from sklearn.linear_model import LogisticRegression
from onnxmltools import convert_sklearn
import onnxruntime as rt

iris = load_iris()
X, y = iris.data, iris.target
X_train, X_test, y_train, y_test = train_test_split(X, y, test_size=0.8, random_state=42)

lr = LogisticRegression(solver="lbfgs", multi_class="multinomial")
lr.fit(X_train, y_train)

print("Test data accuracy of the logistic regressor is {:.2f}".format(lr.score(X_test, y_test)))

onnx_model = convert_sklearn(lr, initial_types=[("input", FloatTensorType([1, 4]))])
with open("iris_sklearn_logreg.onnx", "wb") as f:
    f.write(onnx_model.SerializeToString())
