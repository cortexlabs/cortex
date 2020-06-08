# Networking

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

## API Gateway

APIs are deployed with an internet-facing API Gateway by default (the API Gateway forwards requests to the API load balancer). Each API can be independently configured to not create the API Gateway endpoint by setting `api_gateway: none` in the `networking` field of the [api configuration](api-configuration.md). If the API Gateway endpoint is not created, your API can still be accessed via the API load balancer; `cortex get API_NAME` will show the load balancer endpoint if API Gateway is disabled.

If you'd like to force traffic to go through your public API Gateway endpoint, set `api_load_balancer_scheme: internal` in your [cluster configuration](../cluster-management/config.md) file before creating your cluster.

If you'd like to ensure that an API is not publicly accessible, set `api_load_balancer_scheme: internal` in your [cluster configuration](../cluster-management/config.md) file, and set `api_gateway: none` in the `networking` field of your [api configuration](api-configuration.md). If you do this, you will need to configure [VPC Peering](../guides/vpc-peering.md) to make prediction requests to your APIs.

Here are the common API networking setups:

| Goal                                        | [`api_gateway`](api-configuration.md) | [`api_load_balancer_scheme`](../cluster-management/config.md) | [custom domain](../guides/custom-domain.md)             |
| :---                                        | :---                                  | :---                                                          | :---                                                    |
| public https endpoint (with API Gateway)    | `public`                              | `internal`                                                    | optional                                                |
| public https endpoint (without API Gateway) | `none`                                | `internet-facing`                                             | required (unless clients skip certificate verification) |
| public http endpoint                        | `none`                                | `internet-facing`                                             | optional                                                |
| private https endpoint                      | `none`                                | `internal`                                                    | required (unless clients skip certificate verification) |
| private http endpoint                       | `none`                                | `internal`                                                    | optional                                                |
