# Custom domain

You can use any custom domain (that you own) for your prediction endpoints. For example, you can make your API accessible via `api.example.com/text-generator`. This guide will demonstrate how to create a dedicated subdomain in AWS Route 53 and, if desired, configure your API load balancer to use an SSL certificate provisioned by AWS Certificate Manager.

## Configure DNS

Decide on a subdomain that you want to dedicate to Cortex APIs. For example if your domain is `example.com`, a valid subdomain can be `api.example.com`. This guide will use `cortexlabs.dev` as the example domain and `api.cortexlabs.dev` as the subdomain.

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

## Generate an SSL certificate

You can skip this section (and continue to [add the DNS record](#add-dns-record)) if you don't need an SSL certificate for your custom domain. If you don't use an SSL certificate, you will need to skip certificate verification when making HTTPS requests to your APIs (e.g. `curl -k https://***`), or make HTTP requests instead (e.g. `curl http://***`).

To create an SSL certificate, go to the [ACM console](https://us-west-2.console.aws.amazon.com/acm/home) and click "Get Started" under the "Provision certificates" section.

![](https://user-images.githubusercontent.com/4365343/82202340-c04ac800-98cf-11ea-9472-89dd6d67eb0d.png)

Select "Request a public certificate" and then "Request a certificate".

![](https://user-images.githubusercontent.com/4365343/82202654-3e0ed380-98d0-11ea-8c57-025f0b69c54f.png)

Enter your subdomain and then click "Next".

![](https://user-images.githubusercontent.com/4365343/82224652-1cbedf00-98f2-11ea-912b-466cee2f6e25.png)

Select "DNS validation" and then click "Next".

![](https://user-images.githubusercontent.com/4365343/82205311-66003600-98d4-11ea-90e3-da7e8b0b2b9c.png)

Add tags for searchability (optional) then click "Review".

![](https://user-images.githubusercontent.com/4365343/82206485-52ee6580-98d6-11ea-95a9-1d0ebafc178a.png)

Click "Confirm and request".

![](https://user-images.githubusercontent.com/4365343/82206602-84ffc780-98d6-11ea-9f2f-ce383404ec67.png)

Click "Create record in Route 53". A popup will appear indicating that a Record is going to be added to Route 53. Click "Create" to automatically add the DNS record to your subdomain's hosted zone. Then click "Continue".

![](https://user-images.githubusercontent.com/4365343/82223539-c8ffc600-98f0-11ea-93a2-044aa0c9670d.png)

Wait for the Certificate Status to be "issued". This might take a few minutes.

![](https://user-images.githubusercontent.com/4365343/82209663-a616e700-98db-11ea-95cb-c6efedadb942.png)

Take note of the certificate's ARN. The certificate is ineligible for renewal because it is currently not being used. It will be eligible for renewal once it's used in Cortex.

![](https://user-images.githubusercontent.com/4365343/82222684-9e613d80-98ef-11ea-98c0-5a20b457f062.png)

Add the following field to your cluster configuration:

```yaml
# cluster.yaml

...

ssl_certificate_arn: <ARN of your certificate>
```

Create a Cortex cluster:

```bash
cortex cluster up --config cluster.yaml
```

## Add DNS record

Navigate to your [EC2 Load Balancer console](https://us-west-2.console.aws.amazon.com/ec2/v2/home#LoadBalancers:sort=loadBalancerName) and locate the Cortex API load balancer. You can determine which is the API load balancer by inspecting the `kubernetes.io/service-name` tag.

Take note of the load balancer's name.

![](https://user-images.githubusercontent.com/808475/80142777-961c1980-8560-11ea-9202-40964dbff5e9.png)

Go back to the [Route 53 console](https://console.aws.amazon.com/route53/home#hosted-zones:) and select the hosted zone you created earlier. Click "Create Record Set", and add an Alias record that routes traffic to your Cortex cluster's API load balancer (leave "Name" blank).

![](https://user-images.githubusercontent.com/808475/84083422-6ac97e80-a996-11ea-9679-be37268a2133.png)

## Use your new endpoint

Wait a few minutes to allow the DNS changes to propagate. You may now use your subdomain in place of your API load balancer endpoint in your client. For example, this curl request:

```bash
curl http://a5044e34a352d44b0945adcd455c7fa3-32fa161d3e5bcbf9.elb.us-west-2.amazonaws.com/text-generator -X POST -H "Content-Type: application/json" -d @sample.json
```

Would become:

```bash
# add the `-k` flag or use http:// instead of https:// if you didn't configure an SSL certificate
curl https://api.cortexlabs.dev/text-generator -X POST -H "Content-Type: application/json" -d @sample.json
```

## Debugging connectivity issues

You could run into connectivity issues if you make a request to your API without waiting long enough for your DNS records to propagate after creating them (it usually takes 5-10 mintues). If you are updating existing DNS records, it could take anywhere from a few minutes to 48 hours for the DNS cache to expire (until then, your previous DNS configuration will be used).

To test connectivity, try the following steps:

1. Deploy any api (e.g. examples/pytorch/iris-classifier).
1. Make a GET request to the your api (e.g. `curl https://api.cortexlabs.dev/iris-classifier` or paste the url into your browser).
1. If you run into an error such as `curl: (6) Could not resolve host: api.cortexlabs.dev` wait a few minutes and make the GET request from another device that hasn't made a request to that url in a while. A successful request looks like this:

```text
{"message":"make a prediction by sending a post request to this endpoint with a json payload",...}
```

## Cleanup

Spin down your Cortex cluster.

Delete the hosted zone for your subdomain in the [Route 53 console](https://console.aws.amazon.com/route53/home#hosted-zones:):

![](https://user-images.githubusercontent.com/4365343/82228729-81306d00-98f7-11ea-8570-e9de15f5267f.png)

If you created an SSL certificate, delete it from the [ACM console](https://us-west-2.console.aws.amazon.com/acm/home):

![](https://user-images.githubusercontent.com/4365343/82228835-a624e000-98f7-11ea-92e2-cb4fb0f591e2.png)
