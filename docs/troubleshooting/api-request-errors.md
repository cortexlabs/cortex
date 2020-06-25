# 404 or 503 error responses from API requests

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

When making prediction requests to your API, it's possible to get a `{"message":"Not Found"}` error message (with HTTP status code `404`), or a `no healthy upstream` error message (with HTTP status code `503`). This means that there are currently no live replicas running for your API. This could happen for a few reasons:

1. It's possible that your API is simply not ready yet. You can check the status of your API with `cortex get API_NAME`, and stream the logs with `cortex logs API_NAME`.
1. Your API may have errored. `cortex get API_NAME` will show the status of your API, and you can view the logs with `cortex logs API_NAME`.
1. If `cortex get API_NAME` shows your API's status as "updating" for a while and if `cortex logs API_NAME` doesn't shed any light onto what may be wrong, please see the [API is stuck updating](stuck-updating.md) troubleshooting guide.
