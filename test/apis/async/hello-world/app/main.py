import os

from fastapi import FastAPI
from fastapi.responses import PlainTextResponse

app = FastAPI()
app.response_str = os.getenv("RESPONSE", "hello world")


@app.get("/healthz")
def healthz():
    return PlainTextResponse("ok")


@app.post("/")
def handler():
    return {"message": app.response_str}
