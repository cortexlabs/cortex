import requests
import time
from cortex_internal.lib.log import logger as cortex_logger


class PythonPredictor:
    def __init__(self, config):
        num_success = 0
        num_fail = 0
        for i in range(config["num_requests"]):
            if i > 0:
                time.sleep(config["sleep"])
            response = requests.post(config["endpoint"], json=config["data"], timeout=3600)
            if response.status_code == 200:
                num_success += 1
            else:
                num_fail += 1
                cortex_logger.error(
                    "ERROR",
                    extra={"error": True, "code": response.status_code, "body": response.text},
                )

        cortex_logger.warn(
            "FINISHED",
            extra={"finished": True, "num_success": num_success, "num_fail": num_fail},
        )

    def predict(self, payload):
        return "ok"
