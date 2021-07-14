# API Gateway

This guide shows how set up AWS API Gateway for your Cortex APIs, which is the simplest way to enable HTTPS if a custom domain is not required. See [here](https.md) for how to set up a custom domain with SSL certificates instead.

Please note that one limitation of API Gateway is that there is a 30-second time limit for all requests.

If your API load balancer is internet-facing (which is the default, or you set `api_load_balancer_scheme: internet-facing` in your cluster configuration file before creating your cluster), use the [first section](#internet-facing-load-balancer) of this guide.

If your API load balancer is internal (i.e. you set `api_load_balancer_scheme: internal` in your cluster configuration file before creating your cluster), use the [second section](#internal-load-balancer) of this guide.

## Internet-facing load balancer

_This section applies if your API load balancer is internet-facing (which is the default, or you set `api_load_balancer_scheme: internet-facing` in your cluster configuration file before creating your cluster). If your API load balancer is internal, see the [internal load balancer](#internal-load-balancer) section below._

There are two types of API Gateways that can be used when your load balancer is internet-facing: HTTP and REST. HTTP APIs are faster and less expensive, while REST APIs have more features an allow for more control. See the following section for creating an [HTTP API](#http-api), or skip to the next section for creating a [REST API](#rest-api).

### HTTP API

#### Create an API Gateway

Go to the [API Gateway console](https://console.aws.amazon.com/apigateway/home), select "HTTP API" under "Choose an API type", and click "Build".

![](https://user-images.githubusercontent.com/808475/125668597-35ad8d8e-8b5f-4274-bbb3-62ff47e5d544.png)

Click "Add integration".

![](https://user-images.githubusercontent.com/808475/125668635-f92df672-f516-45e0-a152-538aff901c0d.png)

Click the drop-down menu, and select "HTTP".

![](https://user-images.githubusercontent.com/808475/125668655-78754a51-c77a-4a03-a548-78ab991d1486.png)

Set the "URL endpoint" to `http://API_LOAD_BALANCER_ENDPOINT/{proxy}`. You can get your API load balancer endpoint via `cortex cluster info`; make sure to prepend `http://` and append `/{proxy}`. For example, mine is: `http://aa9f5fdabfa6446aca53a526f59bc3c5-18cd00a628421fa3.elb.us-east-1.amazonaws.com/{proxy}`.

Choose a name for your API (e.g. "cortex"), and click "Next".

![](https://user-images.githubusercontent.com/808475/125668735-b57976c7-f68e-4a28-ab2a-92cfad5b49bb.png)

On the next page, update the pre-populated route's resource path to `/{proxy+}`, and click "Next".

![](https://user-images.githubusercontent.com/808475/125668752-21aef8dd-1e75-41a5-bfc5-1e7157efbe01.png)

Click "Next" again.

![](https://user-images.githubusercontent.com/808475/125668761-8a177f6e-1977-48a2-be84-4b0df0105283.png)

Click "Create".

![](https://user-images.githubusercontent.com/808475/125668770-69fd3fbe-24e4-4a8a-8413-d50c779392c5.png)

Copy your "Invoke URL" for the `$default` stage

![](https://user-images.githubusercontent.com/808475/125668795-b18b564c-7091-432c-a4fd-a5a8bb157946.png)

#### Use your new endpoint

You may now use the "Invoke URL" in place of your API load balancer endpoint in your client. For example, this curl request:

```bash
curl -X POST http://aa9f5fdabfa6446aca53a526f59bc3c5-18cd00a628421fa3.elb.us-east-1.amazonaws.com/hello-world
```

Would become:

```bash
curl -X POST https://nj3f5l96oe.execute-api.us-east-1.amazonaws.com/hello-world
```

#### Cleanup

Delete the API Gateway before spinning down your Cortex cluster:

![](https://user-images.githubusercontent.com/808475/125668816-83bce6bf-cf0e-4835-8d18-d641eb6cdb29.png)

### REST API

#### Create an API Gateway

Go to the [API Gateway console](https://console.aws.amazon.com/apigateway/home), select "REST API" under "Choose an API type", and click "Build".

![](https://user-images.githubusercontent.com/808475/78293216-18269e80-74dd-11ea-9e68-86922c2cbc7c.png)

Select "REST" and "New API", name your API (e.g. "cortex"), select either "Regional" or "Edge optimized" (depending on your preference), and click "Create API".

![](https://user-images.githubusercontent.com/808475/78293434-66d43880-74dd-11ea-92d6-692158171a3f.png)

Select "Actions" > "Create Resource":

![](https://user-images.githubusercontent.com/808475/80154502-8b6b7f80-8574-11ea-9c78-7d9f277bf55b.png)

Select "Configure as proxy resource" and "Enable API Gateway CORS", and click "Create Resource"

![](https://user-images.githubusercontent.com/808475/80154565-ad650200-8574-11ea-8753-808cd35902e2.png)

Select "HTTP Proxy" and set "Endpoint URL" to `http://API_LOAD_BALANCER_ENDPOINT/{proxy}`. You can get your API load balancer endpoint via `cortex cluster info`; make sure to prepend `http://` and append `/{proxy}`. For example, mine is: `http://a9eaf69fd125947abb1065f62de59047-81cdebc0275f7d96.elb.us-west-2.amazonaws.com/{proxy}`.

Leave "Content Handling" set to "Passthrough" and Click "Save".

![](https://user-images.githubusercontent.com/808475/80154735-13ea2000-8575-11ea-83ca-58f182df83c6.png)

Select "Actions" > "Deploy API"

![](https://user-images.githubusercontent.com/808475/80154802-2c5a3a80-8575-11ea-9ab3-de89885fd658.png)

Create a new stage (e.g. "dev") and click "Deploy"

![](https://user-images.githubusercontent.com/808475/80154859-4431be80-8575-11ea-9305-50384b1f9847.png)

Copy your "Invoke URL"

![](https://user-images.githubusercontent.com/808475/80154911-5dd30600-8575-11ea-9682-1a7328783011.png)

#### Use your new endpoint

You may now use the "Invoke URL" in place of your API load balancer endpoint in your client. For example, this curl request:

```bash
curl -X POST http://a9eaf69fd125947abb1065f62de59047-81cdebc0275f7d96.elb.us-west-2.amazonaws.com/hello-world
```

Would become:

```bash
curl -X POST https://31qjv48rs6.execute-api.us-west-2.amazonaws.com/dev/hello-world
```

#### Cleanup

Delete the API Gateway before spinning down your Cortex cluster:

![](https://user-images.githubusercontent.com/808475/80155073-bdc9ac80-8575-11ea-99a1-95c0579da79e.png)

## Internal load balancer

_This section applies if your API load balancer is internal (i.e. you set `api_load_balancer_scheme: internal` in your cluster configuration file before creating your cluster). If your API load balancer is internet-facing, see the [internet-facing load balancer](#internet-facing-load-balancer) section above._

### Create a VPC Link

Navigate to AWS's EC2 Load Balancer dashboard and locate the Cortex API load balancer. You can determine which is the API load balancer by inspecting the `kubernetes.io/service-name` tag:

![](https://user-images.githubusercontent.com/808475/80142777-961c1980-8560-11ea-9202-40964dbff5e9.png)

Take note of the load balancer's name.

Go to the [API Gateway console](https://console.aws.amazon.com/apigateway/home), click "VPC Links" on the left sidebar, and click "Create"

![](https://user-images.githubusercontent.com/808475/80142466-0c6c4c00-8560-11ea-8293-eb5e5572b797.png)

Select "VPC link for REST APIs", name your VPC link (e.g. "cortex"), select the API load balancer, and click "Create".

![](https://user-images.githubusercontent.com/808475/80143027-03c84580-8561-11ea-92de-9ed0a5dfa593.png)

Wait for the VPC link to be created (it will take a few minutes)

![](https://user-images.githubusercontent.com/808475/80144088-bbaa2280-8562-11ea-901b-8520eb253df7.png)

### Create an API Gateway

Go to the [API Gateway console](https://console.aws.amazon.com/apigateway/home), select "REST API" under "Choose an API type", and click "Build"

![](https://user-images.githubusercontent.com/808475/78293216-18269e80-74dd-11ea-9e68-86922c2cbc7c.png)

Select "REST" and "New API", name your API (e.g. "cortex"), select either "Regional" or "Edge optimized" (depending on your preference), and click "Create API"

![](https://user-images.githubusercontent.com/808475/78293434-66d43880-74dd-11ea-92d6-692158171a3f.png)

Select "Actions" > "Create Resource"

![](https://user-images.githubusercontent.com/808475/80141938-3cffb600-855f-11ea-9c1c-132ca4503b7a.png)

Select "Configure as proxy resource" and "Enable API Gateway CORS", and click "Create Resource"

![](https://user-images.githubusercontent.com/808475/80142124-80f2bb00-855f-11ea-8e4e-9413146e0815.png)

Select "VPC Link", select "Use Proxy Integration", choose your newly-created VPC Link, and set "Endpoint URL" to `http://API_LOAD_BALANCER_ENDPOINT/{proxy}`. You can get your API load balancer endpoint via `cortex cluster info`; make sure to prepend `http://` and append `/{proxy}`. For example, mine is: `http://a5044e34a352d44b0945adcd455c7fa3-32fa161d3e5bcbf9.elb.us-west-2.amazonaws.com/{proxy}`. Click "Save"

![](https://user-images.githubusercontent.com/808475/80147407-4f322200-8568-11ea-8ef5-df5164c1375f.png)

Select "Actions" > "Deploy API"

![](https://user-images.githubusercontent.com/808475/80147555-86083800-8568-11ea-86af-1b1e38c9d322.png)

Create a new stage (e.g. "dev") and click "Deploy"

![](https://user-images.githubusercontent.com/808475/80147631-a7692400-8568-11ea-8a09-13dbd50b17b9.png)

Copy your "Invoke URL"

![](https://user-images.githubusercontent.com/808475/80147716-c798e300-8568-11ea-9aef-7dd6fdf4a68a.png)

### Use your new endpoint

You may now use the "Invoke URL" in place of your API load balancer endpoint in your client. For example, this curl request:

```bash
curl -X POST http://a5044e34a352d44b0945adcd455c7fa3-32fa161d3e5bcbf9.elb.us-west-2.amazonaws.com/hello-world
```

Would become:

```bash
curl -X POST https://lrivodooqh.execute-api.us-west-2.amazonaws.com/dev/hello-world
```

### Cleanup

Delete the API Gateway and VPC Link before spinning down your Cortex cluster:

![](https://user-images.githubusercontent.com/808475/80149163-05970680-856b-11ea-9f82-61f4061a3321.png)

![](https://user-images.githubusercontent.com/808475/80149204-1ba4c700-856b-11ea-83f7-9741c78b6b95.png)
