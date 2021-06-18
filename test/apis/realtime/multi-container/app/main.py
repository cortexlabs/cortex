import base64
import requests

from fastapi import FastAPI
from pydantic import BaseModel


class Request(BaseModel):
    image_url: str


app = FastAPI()
app.server_url = "http://localhost:8501/v1/models/resnet50:predict"
app.labels = requests.get(
    "https://storage.googleapis.com/download.tensorflow.org/data/ImageNetLabels.txt"
).text.split("\n")[1:]


@app.get("/healthz")
def healthz():
    return "ok"


@app.post("/")
def text_generator(request: Request):
    # download the image
    dl_request = requests.get(request.image_url, stream=True)
    dl_request.raise_for_status()

    # compose a JSON Predict request (send JPEG image in base64).
    jpeg_bytes = base64.b64encode(dl_request.content).decode("utf-8")
    predict_request = '{"instances" : [{"b64": "%s"}]}' % jpeg_bytes

    # make prediction
    response = requests.post(app.server_url, data=predict_request)
    response.raise_for_status()
    label_id = response.json()["predictions"][0]["classes"]
    return {"image_prediction": app.labels[label_id]}
