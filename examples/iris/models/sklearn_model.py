import numpy as np
from sklearn.datasets import load_iris
from sklearn.model_selection import train_test_split
from sklearn.linear_model import LogisticRegression
from sklearn.preprocessing import StandardScaler
from onnxmltools import convert_sklearn
from onnxconverter_common.data_types import FloatTensorType

iris = load_iris()
X, y = iris.data, iris.target

scaler = StandardScaler()
X = scaler.fit_transform(X)
print("mean:", scaler.mean_)
print("standard deviation:", np.sqrt(scaler.var_))


X_train, X_test, y_train, y_test = train_test_split(X, y, test_size=0.8, random_state=42)

logreg_model = LogisticRegression(solver="lbfgs", multi_class="multinomial")
logreg_model.fit(X_train, y_train)

print("Test data accuracy: {:.2f}".format(logreg_model.score(X_test, y_test)))

# Convert to ONNX model format
onnx_model = convert_sklearn(logreg_model, initial_types=[("input", FloatTensorType([1, 4]))])
with open("sklearn.onnx", "wb") as f:
    f.write(onnx_model.SerializeToString())
