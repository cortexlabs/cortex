import logging

from gunicorn import glogging
from cortex.lib.log import formatter_pid


class CortexGunicornLogger(glogging.Logger):
    def setup(self, cfg):
        super().setup(cfg)
        self._set_handler(self.error_log, cfg.errorlog, formatter_pid)
        self._set_handler(self.access_log, cfg.accesslog, formatter_pid)
