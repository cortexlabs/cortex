# APISplitter configuration

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_


APISplitter is feature which allows you to split traffic between multiple APIs. This can be useful if you want to roll out updated models.


## APISplitter

APISplitter expects the specified APIs to be already deployed. The traffic is routed according to the defined weights. The weights of all APIs need to add up to 100.

```yaml
- name: <string>  # APISplitter name (required)
  kind: APISplitter  # must be "APISplitter", create a APISplitter which routes traffic to multiple SyncAPIs
  networking:
    endpoint: <string>  # the endpoint for the APISplitter (aws only) (default: <api_name>)
    api_gateway: public | none  # whether to create a public API Gateway endpoint for this APISplitter (if not, the load balancer will be accessed directly) (default: public)
  apis:  # list of APIs to use in APISplitter
    - name: <string>  # name of predictor API
      weight: <int>   # proportion of traffic (all APIs add up to 100)
```
