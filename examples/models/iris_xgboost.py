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

iris = load_iris()
X, y = iris.data, iris.target
X_train, X_test, y_train, y_test = train_test_split(X, y, test_size=0.8, random_state=42)

xgb_model = xgb.XGBClassifier()
xgb_model = xgb_model.fit(X_train, y_train)

print("Test data accuracy of the xgb classifier is {:.2f}".format(xgb_model.score(X_test, y_test)))

# Convert to ONNX model format
onnx_model = convert_xgboost(xgb_model, initial_types=[("input", FloatTensorType([1, 4]))])
with open("iris_xgb.onnx", "wb") as f:
    f.write(onnx_model.SerializeToString())
