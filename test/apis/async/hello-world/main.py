import os
from fastapi import FastAPI

app = FastAPI()

response_str = os.getenv("RESPONSE", "hello world")


@app.get("/healthz")
def healthz():
    return "ok"


@app.post("/")
def handler():
    return {"message": response_str}
