import time
from datetime import datetime


class Handler:
    def __init__(self):
        pass

    def handle_post(self, payload, headers):
        print(
            f"{datetime.utcnow().strftime('%Y-%m-%d %H:%M:%S.%f')[:-3]} | {headers['x-request-id']} | request start",
            flush=True,
        )
        start_time = time.time()

        text = payload["image"].file.read()
        print(len(text), flush=True)

        process_time = time.time() - start_time
        print(
            f"{datetime.utcnow().strftime('%Y-%m-%d %H:%M:%S.%f')[:-3]} | {headers['x-request-id']} | request finish: "
            + str(process_time),
            flush=True,
        )

        return "ok"
