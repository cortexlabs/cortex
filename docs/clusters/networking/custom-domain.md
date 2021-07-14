# Custom domain

You can set up DNS to use a custom domain for your Cortex APIs. For example, you can make your API accessible via `api.example.com/hello-world`.

This guide will demonstrate how to create a dedicated subdomain in AWS Route 53. After completing this guide, if you want to enable HTTPS with your custom subdomain, see [these instructions](https.md).

## Configure DNS

Decide on a subdomain that you want to dedicate to Cortex APIs. For example if your domain is `example.com`, a valid subdomain can be `api.example.com`. This guide will use `cortexlabs.dev` as the domain and `api.cortexlabs.dev` as the subdomain.

We will set up a hosted zone on Route 53 to manage the DNS records for the subdomain. Go to the [Route 53 console](https://console.aws.amazon.com/route53/home) and click "Hosted Zones".

![](https://user-images.githubusercontent.com/4365343/82210754-a6b07d00-98dd-11ea-9cec-9f6b07282aa8.png)

Click "Create Hosted Zone" and then enter your subdomain as the domain name for your hosted zone and click "Create".

![](https://user-images.githubusercontent.com/4365343/82211091-4968fb80-98de-11ea-8ec4-8d26d1aea77a.png)

Take note of the values in the NS record.

![](https://user-images.githubusercontent.com/4365343/82211656-386cba00-98df-11ea-8c86-4961082b5f49.png)

Navigate to your root DNS service provider (e.g. Google Domains, AWS Route 53, Go Daddy). Your root DNS service provider is typically the registrar where you purchased your domain (unless you have transferred DNS management elsewhere). The procedure for adding DNS records may vary based on your service provider.

We are going to add an NS (name server) record that specifies that any traffic to your subdomain should use the name servers of your hosted zone in Route 53 for DNS resolution.

`cortexlabs.dev` is managed by Google Domains. The image below is a screenshot for adding a DNS record in Google Domains (your UI may differ based on your DNS service provider).

![](https://user-images.githubusercontent.com/808475/109039458-abcb0580-7681-11eb-8644-76436328687e.png)

## Add DNS record

Navigate to your [EC2 Load Balancer console](https://us-west-2.console.aws.amazon.com/ec2/v2/home#LoadBalancers:sort=loadBalancerName) and locate the Cortex API load balancer. You can determine which is the API load balancer by inspecting the `kubernetes.io/service-name` tag.

Take note of the load balancer's name.

![](https://user-images.githubusercontent.com/808475/80142777-961c1980-8560-11ea-9202-40964dbff5e9.png)

Go back to the [Route 53 console](https://console.aws.amazon.com/route53/home#hosted-zones:) and select the hosted zone you created earlier. Click "Create Record Set", and add an Alias record that routes traffic to your Cortex cluster's API load balancer (leave "Name" blank).

![](https://user-images.githubusercontent.com/808475/84083422-6ac97e80-a996-11ea-9679-be37268a2133.png)

## Debugging connectivity issues

You could run into connectivity issues if you make a request to your API without waiting long enough for your DNS records to propagate after creating them (it usually takes 5-10 minutes). If you are updating existing DNS records, it could take anywhere from a few minutes to 48 hours for the DNS cache to expire (until then, your previous DNS configuration will be used).

To test connectivity, try the following steps:

1. Deploy an api.
1. Make a request to your api (e.g. `curl http://api.cortexlabs.dev/hello-world` or paste the url into your browser if your API supports GET requests).
1. If you run into an error such as `curl: (6) Could not resolve host: api.cortexlabs.dev` wait a few minutes and make the request from another device that hasn't made a request to that url in a while.

## Cleanup

Spin down your Cortex cluster.

Delete the hosted zone for your subdomain in the [Route 53 console](https://console.aws.amazon.com/route53/home#hosted-zones:):

![](https://user-images.githubusercontent.com/4365343/82228729-81306d00-98f7-11ea-8570-e9de15f5267f.png)
