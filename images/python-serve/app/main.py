import time
import logging
from fastapi import FastAPI
from starlette.middleware.base import BaseHTTPMiddleware, RequestResponseEndpoint
from starlette.requests import Request
import uuid
import os
from concurrent.futures import ThreadPoolExecutor
import asyncio

loop = asyncio.get_event_loop()
loop.set_default_executor(ThreadPoolExecutor(max_workers=int(os.environ["THREADS"])))

logger = logging.getLogger("api")

app = FastAPI()


@app.middleware("http")
async def my_middleware(request: Request, call_next):
    pid = os.getpid()
    rid = uuid.uuid4()
    file_id = f"/requests/{pid}{rid}"

    start_time = time.time()
    f = open(file_id, "w")
    f.close()
    logger.info(f"open file: {time.time() - start_time}")

    count = len(os.listdir("/requests"))
    logger.info(f"starting {count}")
    response = await call_next(request)

    logger.error("DONE")

    start_time = time.time()
    os.remove(file_id)
    logger.info(f"remove file: {time.time() - start_time}")

    return response


@app.post("/predict")
async def predict(request: Request):
    print(request)
    logger.info(request)
    body = await request.json()
    print(body)
    print("printing this!")
    logger.error("hi")
    time.sleep(10)
    return {"Hello": "World"}
