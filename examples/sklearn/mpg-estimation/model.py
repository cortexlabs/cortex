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
X = df.drop("mpg", axis=1)
y = df[["mpg"]]

X_train, X_test, y_train, y_test = train_test_split(X, y, test_size=0.2, random_state=1)

model = LinearRegression()
model.fit(X_train, y_train)

mlflow.sklearn.save_model(model, "linreg")
