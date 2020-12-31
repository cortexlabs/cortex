# Setting up HTTPS through the load balancer

This guide assumes you have a custom domain assigned to your API. If you haven't set up custom domain yet, please follow this [guide](../custom_domain/dns.md).

## Generate a certificate for your domain

### Step 1

We are going to create an SSL certificate for your subdomain. Go to the [ACM console](https://us-west-2.console.aws.amazon.com/acm/home) and click "Get Started" under the "Provision certificates" section.

![step 6](https://user-images.githubusercontent.com/4365343/82202340-c04ac800-98cf-11ea-9472-89dd6d67eb0d.png)

### Step 2

Select "Request a public certificate" and then "Request a certificate".

![step 7](https://user-images.githubusercontent.com/4365343/82202654-3e0ed380-98d0-11ea-8c57-025f0b69c54f.png)

### Step 3

Enter your subdomain and then click "Next".

![step 8](https://user-images.githubusercontent.com/4365343/82224652-1cbedf00-98f2-11ea-912b-466cee2f6e25.png)

### Step 4

Select "DNS validation" and then click "Next".

![step 9](https://user-images.githubusercontent.com/4365343/82205311-66003600-98d4-11ea-90e3-da7e8b0b2b9c.png)

### Step 5

Add tags for searchability (optional) then click "Review".

![step 10](https://user-images.githubusercontent.com/4365343/82206485-52ee6580-98d6-11ea-95a9-1d0ebafc178a.png)

### Step 6

Click "Confirm and request".

![step 11](https://user-images.githubusercontent.com/4365343/82206602-84ffc780-98d6-11ea-9f2f-ce383404ec67.png)

### Step 7

Click "Create record in Route 53". A popup will appear indicating that a Record is going to be added to Route 53. Click "Create" to automatically add the DNS record to your subdomain's hosted zone. Then click "Continue".

![step 12](https://user-images.githubusercontent.com/4365343/82223539-c8ffc600-98f0-11ea-93a2-044aa0c9670d.png)

### Step 8

Wait for the Certificate Status to be "issued". This might take a few minutes.

![step 13](https://user-images.githubusercontent.com/4365343/82209663-a616e700-98db-11ea-95cb-c6efedadb942.png)

### Step 9

Take note of the certificate's ARN. The certificate is ineligible for renewal because it is currently not being used. It will be eligible for renewal after it is used in Cortex.

![step 14](https://user-images.githubusercontent.com/4365343/82222684-9e613d80-98ef-11ea-98c0-5a20b457f062.png)

## Configure the API load balancer

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
