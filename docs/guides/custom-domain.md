# Set up a custom domain

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

You can use any custom domain (that you own) for your prediction endpoints. For example, you can make your API accessible via `api.example.com/text-generator`. This guide will demonstrate how to create a dedicated subdomain in AWS Route 53 and use an SSL certificate provisioned by AWS Certificate Manager (ACM).

There are two methods for achieving this, and which method to use depends on whether you're using API Gateway or not (without API Gateway, requests are sent directly to the API load balancer instead). API Gateway is enabled by default.

The first set of steps are the same whether or not you're using API Gateway. If you aren't using API gateway, follow this guide before creating your Cortex cluster.

_note: you must own a domain and be able to modify its DNS records to complete this guide._

## Generate a certificate for your domain

### Step 1

Decide on a subdomain that you want to dedicate to Cortex APIs. For example if your domain is `example.com`, a valid subdomain can be `api.example.com`.

This guide will use `cortexlabs.dev` as the example domain and `api.cortexlabs.dev` as the subdomain.

### Step 2

We will set up a hosted zone on Route 53 to manage the DNS records for the subdomain. Go to the [Route 53 console](https://console.aws.amazon.com/route53/home) and click "Hosted Zones".

![step 2](https://user-images.githubusercontent.com/4365343/82210754-a6b07d00-98dd-11ea-9cec-9f6b07282aa8.png)

### Step 3

Click "Create Hosted Zone" and then enter your subdomain as the domain name for your hosted zone and click "Create".

![step 3](https://user-images.githubusercontent.com/4365343/82211091-4968fb80-98de-11ea-8ec4-8d26d1aea77a.png)

### Step 4

Take note of the values in the NS record.

![step 4](https://user-images.githubusercontent.com/4365343/82211656-386cba00-98df-11ea-8c86-4961082b5f49.png)

### Step 5

Navigate to your root DNS service provider (e.g. Google Domains, AWS Route 53, Go Daddy). Your root DNS service provider is typically the registrar where you purchased your domain (unless you have transferred DNS management elsewhere). The procedure for adding DNS records may vary based on your service provider.

We are going to add an NS (name server) record that specifies that any traffic to your subdomain should use the name servers of your hosted zone in Route 53 for DNS resolution.

`cortexlabs.dev` is managed by Google Domains. The image below is a screenshot for adding a DNS record in Google Domains (your UI may differ based on your DNS service provider).

![step 5](https://user-images.githubusercontent.com/4365343/82211959-bcbf3d00-98df-11ea-834d-692b3bcf9332.png)

### Step 6

We are now going to create an SSL certificate for your subdomain. Go to the [ACM console](https://us-west-2.console.aws.amazon.com/acm/home) and click "Get Started" under the "Provision certificates" section.

![step 6](https://user-images.githubusercontent.com/4365343/82202340-c04ac800-98cf-11ea-9472-89dd6d67eb0d.png)

### Step 7

Select "Request a public certificate" and then "Request a certificate".

![step 7](https://user-images.githubusercontent.com/4365343/82202654-3e0ed380-98d0-11ea-8c57-025f0b69c54f.png)

### Step 8

Enter your subdomain and then click "Next".

![step 8](https://user-images.githubusercontent.com/4365343/82224652-1cbedf00-98f2-11ea-912b-466cee2f6e25.png)

### Step 9

Select "DNS validation" and then click "Next".

![step 9](https://user-images.githubusercontent.com/4365343/82205311-66003600-98d4-11ea-90e3-da7e8b0b2b9c.png)

### Step 10

Add tags for searchability (optional) then click "Review".

![step 10](https://user-images.githubusercontent.com/4365343/82206485-52ee6580-98d6-11ea-95a9-1d0ebafc178a.png)

### Step 11

Click "Confirm and request".

![step 11](https://user-images.githubusercontent.com/4365343/82206602-84ffc780-98d6-11ea-9f2f-ce383404ec67.png)

### Step 12

Click "Create record in Route 53". A popup will appear indicating that a Record is going to be added to Route 53. Click "Create" to automatically add the DNS record to your subdomain's hosted zone. Then click "Continue".

![step 12](https://user-images.githubusercontent.com/4365343/82223539-c8ffc600-98f0-11ea-93a2-044aa0c9670d.png)

### Step 13

Wait for the Certificate Status to be "issued". This might take a few minutes.

![step 13](https://user-images.githubusercontent.com/4365343/82209663-a616e700-98db-11ea-95cb-c6efedadb942.png)

### Step 14

Take note of the certificate's ARN. The certificate is ineligible for renewal because it is currently not being used. It will be eligible for renewal after it is used in Cortex.

![step 14](https://user-images.githubusercontent.com/4365343/82222684-9e613d80-98ef-11ea-98c0-5a20b457f062.png)

If you are using API Gateway, continue to the next section to [configure API Gateway](#configure-api-gateway). Otherwise (i.e. clients connect directly to your API load balancer) skip to [configure the API load balancer](#configure-the-api-load-balancer).

## Configure API Gateway

_If you aren't using API Gateway (i.e. clients connect directly to your API load balancer), skip this section and [configure the API load balancer](#configure-the-api-load-balancer) instead._

### Step 1

Navigate to the [API Gateway console](https://us-west-2.console.aws.amazon.com/apigateway) (make sure that the region in top right matches your Cortex region). Click "Custom domain names" and then click "Create".

![step 1](https://user-images.githubusercontent.com/808475/84082403-a105fe80-a994-11ea-8015-0df9aeef07b1.png)

### Step 2

Type in the name of your domain and choose the Regional endpoint type, TLS 1.2, and the ACM certificate that you created earlier. Then click "Create".

![step 2](https://user-images.githubusercontent.com/808475/84082448-b8dd8280-a994-11ea-9ef7-6bb33b3a403b.png)

### Step 3

This should take you back to the custom domains page, and you should see your new custom domain. Make sure your API is selected. Take note of the API Gateway domain name (we will use this later), and click "Configure API mappings".

![step 3](https://user-images.githubusercontent.com/808475/84084960-62267780-a999-11ea-8a6b-4be9cfca2a9c.png)

### Step 4

Click "Add new mapping".

![step 4](https://user-images.githubusercontent.com/808475/84082516-d7dc1480-a994-11ea-9cae-d4fb1ac1e767.png)

### Step 5

Select your API Gateway, choose the "$default" stage, and lave "Path" blank. Click Save.

![step 5](https://user-images.githubusercontent.com/808475/84082553-e5919a00-a994-11ea-9bc3-cf5f9eb869eb.png)

### Step 6

Go back to the [Route 53 console](https://console.aws.amazon.com/route53/home#hosted-zones:) and select the hosted zone you created earlier. Click "Create Record Set", and add an Alias record that routes traffic to your Cortex cluster's API Gateway (the target name should match the "API Gateway domain name" show in step 3). Leave "Name" blank.

![step 6](https://user-images.githubusercontent.com/808475/84083366-54232780-a996-11ea-9bc6-2c9945a160d4.png)

Proceed to [using your new endpoint](#using-your-new-endpoint).

## Configure the API load balancer

_If you are using API Gateway, this section does not apply; follow the instructions above to [configure API Gateway](#configure-api-gateway) instead._

### Step 1

Add the following field to your cluster configuration:

```yaml
# cluster.yaml

...

ssl_certificate_arn: <ARN of your certificate>
```

and then create a Cortex cluster.

```bash
$ cortex cluster up --config cluster.yaml
```

### Step 2

After your cluster has been created, navigate to your [EC2 Load Balancer console](https://us-west-2.console.aws.amazon.com/ec2/v2/home#LoadBalancers:sort=loadBalancerName) and locate the Cortex API load balancer. You can determine which is the API load balancer by inspecting the `kubernetes.io/service-name` tag.

Take note of the load balancer's name.

![step 2](https://user-images.githubusercontent.com/808475/80142777-961c1980-8560-11ea-9202-40964dbff5e9.png)

### Step 3

Go back to the [Route 53 console](https://console.aws.amazon.com/route53/home#hosted-zones:) and select the hosted zone you created earlier. Click "Create Record Set", and add an Alias record that routes traffic to your Cortex cluster's API load balancer (leave "Name" blank).

![step 3](https://user-images.githubusercontent.com/808475/84083422-6ac97e80-a996-11ea-9679-be37268a2133.png)

## Using your new endpoint

Wait a few minutes to allow the DNS changes to propagate. You may now use your subdomain in place of your API load balancer endpoint in your client. For example, this curl request:

```bash
curl http://a5044e34a352d44b0945adcd455c7fa3-32fa161d3e5bcbf9.elb.us-west-2.amazonaws.com/text-generator -X POST -H "Content-Type: application/json" -d @sample.json
```

Would become:

```bash
# replace loadbalancer url with your subdomain
curl https://api.cortexlabs.dev/text-generator -X POST -H "Content-Type: application/json" -d @sample.json
```

## Debugging connectivity issues

You could run into connectivity issues if you make a request to your API without waiting long enough for your DNS records to propagate after creating them (it usually takes 5-10 mintues). If you are updating existing DNS records, it could take anywhere from a few minutes to 48 hours for the DNS cache to expire (until then, your previous DNS configuration will be used).

To test connectivity, try the following steps:

1. Deploy any api (e.g. examples/pytorch/iris-classifier).
1. Make an HTTPS GET request to the your api e.g. `curl https://api.cortexlabs.dev/iris-classifier` or enter the url in your browser.
1. If you run into an error such as `curl: (6) Could not resolve host: api.cortexlabs.dev` wait a few minutes and make the HTTPS Get request from another device that hasn't made a request to that url in a while. A successful request looks like this:

```text
{"message":"make a prediction by sending a post request to this endpoint with a json payload",...}
```

## Cleanup

Spin down your Cortex cluster.

Delete the hosted zone for your subdomain in the [Route 53 console](https://console.aws.amazon.com/route53/home#hosted-zones:):

![delete hosted zone](https://user-images.githubusercontent.com/4365343/82228729-81306d00-98f7-11ea-8570-e9de15f5267f.png)

Delete your certificate from the [ACM console](https://us-west-2.console.aws.amazon.com/acm/home):

![delete certificate](https://user-images.githubusercontent.com/4365343/82228835-a624e000-98f7-11ea-92e2-cb4fb0f591e2.png)
