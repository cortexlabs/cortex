# Custom Domain

You can use any custom domain (that you own) for your prediction endpoints. 
For example, you can make your API accessible via `api.example.com/text-generator`. 
This guide will demonstrate how to create a dedicated subdomain in 
AWS Route 53 and use an SSL certificate provisioned by AWS Certificate Manager (ACM).

There are two methods for achieving this, and which method to use depends on whether you're 
using API Gateway or not (without API Gateway, requests are sent directly to the API 
load balancer instead). API Gateway needs to be setup first in order 

Follow one of the guides below to setup a custom domain for your API:

- [Without API Gateway](dns.md)
- [With API Gateway](api-gateway.md)
