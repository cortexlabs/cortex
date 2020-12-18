import mlflow.sklearn
import pandas as pd
import numpy as np
from sklearn.linear_model import LinearRegression
from sklearn.model_selection import train_test_split


df = pd.read_csv(
    "https://www.uio.no/studier/emner/sv/oekonomi/ECON4150/v16/statacourse/datafiles/auto.csv"
)
df = df.replace("?", np.nan)
df = df.dropna()
df = df.drop(["name", "origin", "year"], axis=1)  # drop categorical variables for simplicity
data = df.drop("mpg", axis=1)
labels = df[["mpg"]]

training_data, test_data, training_labels, test_labels = train_test_split(data, labels)
model = LinearRegression()
model.fit(training_data, training_labels)
accuracy = model.score(test_data, test_labels)
print("accuracy: {:.2f}".format(accuracy))

mlflow.sklearn.save_model(model, "linreg")
