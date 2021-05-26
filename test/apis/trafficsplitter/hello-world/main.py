import os
import time
from fastapi import FastAPI

app = FastAPI()

response_str = os.getenv("RESPONSE", "hello world")


@app.get("/healthz")
def healthz():
    return "ok"


@app.post("/")
def post_handler():
    return response_str
