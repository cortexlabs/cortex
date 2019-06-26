"""
Requirements.txt

onnxmltools
scikit-learn
xgboost
"""
import numpy as np
import xgboost as xgb
from sklearn.datasets import load_iris
from sklearn.model_selection import train_test_split
from onnxmltools.convert import convert_xgboost
from onnxconverter_common.data_types import FloatTensorType
import onnxruntime as rt

iris = load_iris()
X, y = iris.data, iris.target
X_train, X_test, y_train, y_test = train_test_split(X, y, test_size=0.8, random_state=42)

xgb_clf = xgb.XGBClassifier()
xgb_clf = xgb_clf.fit(X_train, y_train)

print("Test data accuracy of the xgb classifier is {:.2f}".format(xgb_clf.score(X_test, y_test)))

onnx_model = convert_xgboost(xgb_clf, initial_types=[("input", FloatTensorType([1, 4]))])
with open("iris_xgb.onnx", "wb") as f:
    f.write(onnx_model.SerializeToString())
