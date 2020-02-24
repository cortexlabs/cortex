import time
import logging
from fastapi import FastAPI
from starlette.middleware.base import BaseHTTPMiddleware, RequestResponseEndpoint
from starlette.requests import Request

logger = logging.getLogger("api")

app = FastAPI()


@app.middleware("http")
async def my_middleware(request: Request, call_next):
    logger.error("HERE")

    response = await call_next(request)

    logger.error("DONE")

    return response


@app.post("/predict")
def predict():
    time.sleep(120)
    return {"Hello": "World"}
