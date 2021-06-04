import time

from fastapi import FastAPI
from fastapi.responses import PlainTextResponse

app = FastAPI()


@app.get("/healthz")
def healthz():
    return PlainTextResponse("ok")


@app.post("/")
def sleep(sleep: float = 0):
    time.sleep(sleep)
    return PlainTextResponse("ok")
