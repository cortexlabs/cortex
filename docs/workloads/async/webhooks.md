# Webhooks

Polling for requests can be very resource intensive, and does not guarantee you the result as soon as it is ready. In
order to overcome this problem, we can use webhooks.

A webhook is a request that is sent to a URL known in advance when an event occurs. In our case, the event is a workload
completion or failure, and the URL known in advance is some other service that we have already running.

## Example

Below is a guideline for implementing webhooks for an `AsyncAPI` workload.

```python
import time
from datetime import datetime

import requests

STATUS_COMPLETED = "completed"
STATUS_FAILED = "failed"


class PythonPredictor:
    def __init__(self, config):
        self.webhook_url = config["webhook_url"]  # the webhook url is passed in the config

    def predict(self, payload, request_id):
        try:
            time.sleep(60)  # simulates a long workload
            self.send_report(request_id, STATUS_COMPLETED, result={"data": "hello"})
        except Exception as err:
            self.send_report(request_id, STATUS_FAILED)
            raise err  # the original exception should still be raised!

    # this is a utility method
    def send_report(self, request_id, status, result=None):
        response = {"id": request_id, "status": status}

        if result is not None and status == STATUS_COMPLETED:
            timestamp = datetime.utcnow().isoformat()
            response.update({"result": result, "timestamp": timestamp})

        try:
            requests.post(url=self.webhook_url, json=response)
        except Exception:
            pass
```

## Development

For development purposes, you can use a utility website such as https://webhook.site/ to validate that your webhook
setup is working as intended.
