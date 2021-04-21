import time


class Handler:
    def __init__(self, config):
        pass

    def handle_post(self, payload, query_params):
        time.sleep(float(query_params.get("sleep", 0)))
        return "ok"
