# Webhooks

Polling for requests can be resource intensive, and does not guarantee that you will have the result as soon as it is ready. In
order to overcome this problem, we can use webhooks.

A webhook is a request that is sent to a URL known in advance when an event occurs. In our case, the event is a workload
completion or failure, and the URL known in advance is some other service that we already have running.

## Example

Below is an example implementing webhooks for an `AsyncAPI` workload using FastAPI.

```python
import os
import time
import requests
from datetime import datetime
from fastapi import FastAPI, Header

STATUS_COMPLETED = "completed"
STATUS_FAILED = "failed"

webhook_url = os.getenv("WEBHOOK_URL")  # the webhook url is set as an environment variable

app = FastAPI()


@app.post("/")
async def handle(x_cortex_request_id=Header(None)):
    try:
        time.sleep(60)  # simulates a long workload
        send_report(x_cortex_request_id, STATUS_COMPLETED, result={"data": "hello"})
    except Exception as err:
        send_report(x_cortex_request_id, STATUS_FAILED)
        raise err  # the original exception should still be raised


def send_report(request_id, status, result=None):
    response = {"id": request_id, "status": status}

    if result is not None and status == STATUS_COMPLETED:
        timestamp = datetime.utcnow().isoformat()
        response.update({"result": result, "timestamp": timestamp})

    try:
        requests.post(url=webhook_url, json=response)
    except Exception:
        pass
```

## Development

For development purposes, you can use a utility website such as https://webhook.site/ to validate that your webhook
setup is working as intended.
