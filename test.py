import threading
import time


class Ticker:
    def __init__(self, interval, func, *args, **kwargs):
        self.interval = interval
        self.func = func
        self.args = args
        self.kwargs = kwargs
        self.generator = self.func(*self.args, **self.kwargs)

    def start(self):
        self.timer = threading.Timer(self.interval, self.start)
        self.timer.start()
        next(self.generator)

    def stop(self):
        self.timer.cancel()


def testing(queue_url, receipt_handle, initial_offset, interval, *args, **kwargs):
    new_timeout = initial_offset + interval
    while True:
        yield
        print(f"{queue_url} {receipt_handle} {initial_offset} {interval} {new_timeout}")
        new_timeout += interval


renewer = Ticker(5, testing, "sqs://URL", "receipt_handle", 90, 5)

renewer.start()

time.sleep(30)
renewer.stop()
print("stopped")
time.sleep(15)
print("confirmed")
