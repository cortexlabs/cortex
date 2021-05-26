import os, time

from fastapi import FastAPI

app = FastAPI()
record_only = os.getenv("RECORD_ONLY", "false")


@app.get("/healthz")
def healthz():
    return "ok"


@app.post("/")
def post_handler():
    if record_only == "true":
        print(f"recording hello-world request at {time.time()} time")
    else:
        print(f"processing hello-world request at {time.time()} time")
