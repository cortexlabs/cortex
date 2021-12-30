import time
import asyncio
from datetime import datetime

from fastapi import FastAPI, Header, Request, File, UploadFile
from fastapi.responses import PlainTextResponse

app = FastAPI()


@app.middleware("http")
async def add_process_time_header(request: Request, call_next):
    if "x-request-id" in request.headers:
        request_id = request.headers["x-request-id"]
    else:
        request_id = "0"

    print(
        f"{datetime.utcnow().strftime('%Y-%m-%d %H:%M:%S.%f')[:-3]} | {request_id} | middleware start",
        flush=True,
    )
    start_time = time.time()
    response = await call_next(request)
    process_time = time.time() - start_time
    print(
        f"{datetime.utcnow().strftime('%Y-%m-%d %H:%M:%S.%f')[:-3]} | {request_id} | middleware finish: {str(process_time)}",
        flush=True,
    )
    return response


@app.get("/healthz")
def healthz():
    return PlainTextResponse("ok")


@app.post("/")
async def sleep(sleep: float = 0, x_request_id: str = Header(None), image: UploadFile = File(...)):
    print(
        f"{datetime.utcnow().strftime('%Y-%m-%d %H:%M:%S.%f')[:-3]} | {x_request_id} | request start",
        flush=True,
    )
    start_time = time.time()
    image = await image.read()
    process_time = time.time() - start_time
    print(
        f"{datetime.utcnow().strftime('%Y-%m-%d %H:%M:%S.%f')[:-3]} | {x_request_id} | downloaded image: "
        + str(process_time),
        flush=True,
    )
    # time.sleep(sleep)
    await asyncio.sleep(sleep)
    process_time = time.time() - start_time
    print(
        f"{datetime.utcnow().strftime('%Y-%m-%d %H:%M:%S.%f')[:-3]} | {x_request_id} | request finish: "
        + str(process_time),
        flush=True,
    )
    return PlainTextResponse("ok")


# r=$((1 + $RANDOM % 100)); echo "request ID: $r"; SECONDS=0; curl -X POST -H "x-request-id: $r" -F image=@wp.jpg http://ad14cde85e57748ff9c384a32617133f-9335e56d3708bc0a.elb.us-west-2.amazonaws.com/sleep?sleep=1 --limit-rate 4k; echo "$SECONDS seconds"
