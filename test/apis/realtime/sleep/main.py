import time
from typing import Optional

from fastapi import FastAPI
from pydantic import BaseModel

class Request(BaseModel):
    sleep_val: Optional[float]

app = FastAPI()

@app.get("/healthz")
def healthz():
    return "ok"

@app.post("/")
def sleep(request: Request):
    sleep_val = 1
    if request.sleep_val and request.sleep_val > 0:
        sleep_val = request.sleep_val
    time.sleep(sleep_val)
