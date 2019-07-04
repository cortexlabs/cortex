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
from sklearn.linear_model import LogisticRegression
from onnxconverter_common.data_types import FloatTensorType
from onnxmltools import convert_sklearn

iris = load_iris()
X, y = iris.data, iris.target
X_train, X_test, y_train, y_test = train_test_split(X, y, test_size=0.8, random_state=42)

logreg_model = LogisticRegression(solver="lbfgs", multi_class="multinomial")
logreg_model.fit(X_train, y_train)

print("Test data accuracy: {:.2f}".format(logreg_model.score(X_test, y_test)))


# Convert to ONNX model format
onnx_model = convert_sklearn(logreg_model, initial_types=[("input", FloatTensorType([1, 4]))])
with open("iris_sklearn_logreg.onnx", "wb") as f:
    f.write(onnx_model.SerializeToString())
