import os

from fastapi import FastAPI
from fastapi.responses import PlainTextResponse

app = FastAPI()
app.response_str = os.getenv("RESPONSE", "hello world")


@app.get("/healthz")
def healthz():
    return PlainTextResponse("ok")


@app.post("/")
def post_handler():
    return PlainTextResponse(app.response_str)
