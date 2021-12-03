from flask import Flask, request
from datetime import datetime
import time

app = Flask(__name__)


@app.route("/", methods=["POST"])
def hello_world():
    print(
        f"{datetime.utcnow().strftime('%Y-%m-%d %H:%M:%S.%f')[:-3]} | {request.headers.get('x-request-id')} | request start",
        flush=True,
    )
    start_time = time.time()

    file = request.files["image"]
    # print(file.content_length, flush=True)

    process_time = time.time() - start_time
    print(
        f"{datetime.utcnow().strftime('%Y-%m-%d %H:%M:%S.%f')[:-3]} | {request.headers.get('x-request-id')} | request finish: "
        + str(process_time),
        flush=True,
    )
    return "Hello, Docker!"
