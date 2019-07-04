"""
Requirements.txt

scikit-learn
keras
keras2onnx
tensorflow
onnxruntime
"""
import numpy as np
from sklearn.datasets import load_iris
from sklearn.model_selection import train_test_split
from keras.models import Sequential
from keras.layers import Dense
from keras.utils import np_utils
import onnxruntime as rt
import keras2onnx

iris = load_iris()
X, y = iris.data, iris.target
y = np_utils.to_categorical(y)
X_train, X_test, y_train, y_test = train_test_split(X, y, test_size=0.2, random_state=42)

model = Sequential(name="iris")
model.add(Dense(30, input_dim=4, activation="relu", name="input"))
model.add(Dense(3, activation="softmax", name="last"))

model.compile(loss="categorical_crossentropy", optimizer="adam", metrics=["accuracy"])

model.fit(X_train, y_train, epochs=100)

scores = model.evaluate(X_test, y_test)
print("\n%s: %.2f%%" % (model.metrics_names[1], scores[1] * 100))

onnx_model = keras2onnx.convert_keras(model)

with open("iris_keras.onnx", "wb") as f:
    f.write(onnx_model.SerializeToString())
