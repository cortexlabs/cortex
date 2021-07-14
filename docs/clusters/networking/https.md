# Setting up HTTPS

This guide shows how to support HTTPS traffic to Cortex APIs via a custom domain. It is also possible to use AWS API Gateway to enable HTTPS without using your own domain (see [here](api-gateway.md) for instructions).

In order to create a valid SSL certificate for your domain, you must have the ability to configure DNS to satisfy the DNS challenges which prove that you own the domain. This guide assumes that you are using a Route 53 hosted zone to manage a subdomain. Follow this [guide](./custom-domain.md) to set up a subdomain managed by a Route 53 hosted zone.

## Generate an SSL certificate

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

## Create or update your cluster

Add the following field to your cluster configuration:

```yaml
# cluster.yaml

...

ssl_certificate_arn: <ARN of your certificate>
```

Create a cluster:

```bash
cortex cluster up cluster.yaml
```

Or update an existing cluster:

```bash
cortex cluster configure cluster.yaml
```

## Use your new endpoint

Wait a few minutes to allow the DNS changes to propagate. You may now use your subdomain in place of your API load balancer endpoint in your client. For example, this curl request:

```bash
curl http://a5044e34a352d44b0945adcd455c7fa3-32fa161d3e5bcbf9.elb.us-west-2.amazonaws.com/hello-world -X POST -H "Content-Type: application/json" -d @sample.json
```

Would become:

```bash
# add the `-k` flag or use http:// instead of https:// if you didn't configure an SSL certificate
curl https://api.cortexlabs.dev/hello-world -X POST -H "Content-Type: application/json" -d @sample.json
```

## Cleanup

Spin down your Cortex cluster.

If you created an SSL certificate, delete it from the [ACM console](https://us-west-2.console.aws.amazon.com/acm/home):

![](https://user-images.githubusercontent.com/4365343/82228835-a624e000-98f7-11ea-92e2-cb4fb0f591e2.png)
