from cortex_internal.lib.log import logger as cortex_logger


class PythonPredictor:
    def __init__(self, config):
        pass

    def predict(self, payload):
        cortex_logger.info("received payload", extra={"payload": payload})
        return payload
