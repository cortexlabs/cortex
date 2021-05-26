import time

from fastapi import FastAPI

app = FastAPI()


@app.get("/healthz")
def healthz():
    return "ok"


@app.post("/")
def sleep(sleep: float = 0):
    time.sleep(sleep)
