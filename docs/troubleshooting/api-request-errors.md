# 404 or 503 error responses from API requests

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

When making prediction requests to your API, it's possible to get a `{"message":"Not Found"}` error message (with HTTP status code `404`), or a `no healthy upstream` error message (with HTTP status code `503`). This means that there are currently no live replicas running for your API. This could happen for a few reasons:

1. It's possible that your API is simply not ready yet. You can check the status of your API with `cortex get API_NAME`, and stream the logs with `cortex logs API_NAME`.
1. Your API may have errored during initialization or while responding to a previous request. `cortex get API_NAME` will show the status of your API, and you can view the logs with `cortex logs API_NAME`.
1. If `cortex get API_NAME` shows your API's status as "updating" for a while and if `cortex logs API_NAME` doesn't shed any light onto what may be wrong, please see the [API is stuck updating](stuck-updating.md) troubleshooting guide.

It is also possible to receive a `{"message":"Service Unavailable"}` error message (with HTTP status code `503`) if you are using an API Gateway endpoint for your API and if your request exceeds API Gateway's 29 second timeout (API Gateway is used by default, and if you're using API gateway, your API's endpoint will start with `https://<id>.execute-api.<region>.amazonaws.com`). To confirm this is the issue: modify your `predict()` function to immediately return e.g. "ok", re-deploy your API, wait for it to be live, and try making a request. If your API is timing out, you can either modify your `predict()` function to take less time or run on faster hardware (e.g. GPUs), or you can disable API Gateway for your API by setting `api_gateway: none` in the `networking` field of the [api configuration](api-configuration.md) (see [networking](../deployments/networking.md) for more details).
