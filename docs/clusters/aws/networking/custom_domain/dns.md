# Custom Domain Without API Gateway

## Configure Domain Name Servers (DNS)

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

## Add the Load Balancer to the DNS Records

### Step 1

Navigate to your [EC2 Load Balancer console](https://us-west-2.console.aws.amazon.com/ec2/v2/home#LoadBalancers:sort=loadBalancerName) and locate the Cortex API load balancer. You can determine which is the API load balancer by inspecting the `kubernetes.io/service-name` tag.

Take note of the load balancer's name.

![step 2](https://user-images.githubusercontent.com/808475/80142777-961c1980-8560-11ea-9202-40964dbff5e9.png)

### Step 2

Go back to the [Route 53 console](https://console.aws.amazon.com/route53/home#hosted-zones:) and select the hosted zone you created earlier. Click "Create Record Set", and add an Alias record that routes traffic to your Cortex cluster's API load balancer (leave "Name" blank).

![step 3](https://user-images.githubusercontent.com/808475/84083422-6ac97e80-a996-11ea-9679-be37268a2133.png)

## Using your new endpoint

Wait a few minutes to allow the DNS changes to propagate. You may now use your subdomain in place of your API load balancer endpoint in your client. For example, this curl request:

```bash
curl http://a5044e34a352d44b0945adcd455c7fa3-32fa161d3e5bcbf9.elb.us-west-2.amazonaws.com/text-generator -X POST -H "Content-Type: application/json" -d @sample.json
```

Would become:

```bash
# replace load balancer url with your subdomain
curl http://api.cortexlabs.dev/text-generator -X POST -H "Content-Type: application/json" -d @sample.json
```
