# REST API Gateway

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

When `api_gateway: public` is set in your API's `networking` configuration (which is the default setting), Cortex will create an "HTTP" API Gateway in AWS for your API (see the [networking docs](../deployments/networking.md) for more information).

However, there may be situations where you need to use AWS's "REST" API Gateway, e.g. to enforce IAM-based auth. Until [#1197](https://github.com/cortexlabs/cortex/issues/1197) is resolved, a REST API Gateway can be used by following these steps.

If your API load balancer is internet-facing (which is the default, or you explicitly set `api_load_balancer_scheme: internet-facing` in your cluster configuration file before creating your cluster), use the [first section](#if-your-api-load-balancer-is-internet-facing) of this guide.

If your API load balancer is internal (i.e. you set `api_load_balancer_scheme: internal` in your cluster configuration file before creating your cluster), use the [second section](#if-your-api-load-balancer-is-internal) of this guide.

## If your API load balancer is internet-facing

### Step 1

Disable the default API Gateway:

* If you haven't created your cluster yet, you can set `api_gateway: none` in your [cluster configuration file](install.md) before creating your cluster.
* If you have already created your cluster, you can set `api_gateway: none` in the `networking` field of your [Realtime API configuration](../deployments/realtime-api/api-configuration.md) and/or [Batch API configuration](../deployments/batch-api/api-configuration.md), and then re-deploy your API.

### Step 2

Go to the [API Gateway console](https://console.aws.amazon.com/apigateway/home), select "REST API" under "Choose an API type", and click "Build"

![step 1](https://user-images.githubusercontent.com/808475/78293216-18269e80-74dd-11ea-9e68-86922c2cbc7c.png)

### Step 3

Select "REST" and "New API", name your API (e.g. "cortex"), select either "Regional" or "Edge optimized" (depending on your preference), and click "Create API"

![step 2](https://user-images.githubusercontent.com/808475/78293434-66d43880-74dd-11ea-92d6-692158171a3f.png)

### Step 4

Select "Actions" > "Create Resource"

![step 3](https://user-images.githubusercontent.com/808475/80154502-8b6b7f80-8574-11ea-9c78-7d9f277bf55b.png)

### Step 5

Select "Configure as proxy resource" and "Enable API Gateway CORS", and click "Create Resource"

![step 4](https://user-images.githubusercontent.com/808475/80154565-ad650200-8574-11ea-8753-808cd35902e2.png)

### Step 6

Select "HTTP Proxy" and set "Endpoint URL" to "http://<BASE_API_ENDPOINT>/{proxy}". You can get your base API endpoint via `cortex cluster info`; make sure to prepend `http://` and append `/{proxy}`. For example, mine is: `http://a9eaf69fd125947abb1065f62de59047-81cdebc0275f7d96.elb.us-west-2.amazonaws.com/{proxy}`.

Leave "Content Handling" set to "Passthrough" and Click "Save".

![step 5](https://user-images.githubusercontent.com/808475/80154735-13ea2000-8575-11ea-83ca-58f182df83c6.png)

### Step 7

Select "Actions" > "Deploy API"

![step 6](https://user-images.githubusercontent.com/808475/80154802-2c5a3a80-8575-11ea-9ab3-de89885fd658.png)

### Step 8

Create a new stage (e.g. "dev") and click "Deploy"

![step 7](https://user-images.githubusercontent.com/808475/80154859-4431be80-8575-11ea-9305-50384b1f9847.png)

### Step 9

Copy your "Invoke URL"

![step 8](https://user-images.githubusercontent.com/808475/80154911-5dd30600-8575-11ea-9682-1a7328783011.png)

### Using your new endpoint

You may now use the "Invoke URL" in place of your APIs endpoint in your client. For example, this curl request:

```bash
curl http://a9eaf69fd125947abb1065f62de59047-81cdebc0275f7d96.elb.us-west-2.amazonaws.com/iris-classifier -X POST -H "Content-Type: application/json" -d @sample.json
```

Would become:

```bash
curl https://31qjv48rs6.execute-api.us-west-2.amazonaws.com/dev/iris-classifier -X POST -H "Content-Type: application/json" -d @sample.json
```

### Cleanup

Delete the API Gateway before spinning down your Cortex cluster:

![delete api gateway](https://user-images.githubusercontent.com/808475/80155073-bdc9ac80-8575-11ea-99a1-95c0579da79e.png)

## If your API load balancer is internal

### Step 1

Disable the default API Gateway:

* If you haven't created your cluster yet, you can set `api_gateway: none` in your [cluster configuration file](install.md) before creating your cluster.
* If you have already created your cluster, you can set `api_gateway: none` in the `networking` field of your [Realtime API configuration](../deployments/realtime-api/api-configuration.md) and/or [Batch API configuration](../deployments/batch-api/api-configuration.md), and then re-deploy your API.

### Step 2

Navigate to AWS's EC2 Load Balancer dashboard and locate the Cortex API load balancer. You can determine which is the API load balancer by inspecting the `kubernetes.io/service-name` tag:

![step 1](https://user-images.githubusercontent.com/808475/80142777-961c1980-8560-11ea-9202-40964dbff5e9.png)

Take note of the load balancer's name.

### Step 3

Go to the [API Gateway console](https://console.aws.amazon.com/apigateway/home), click "VPC Links" on the left sidebar, and click "Create"

![step 2](https://user-images.githubusercontent.com/808475/80142466-0c6c4c00-8560-11ea-8293-eb5e5572b797.png)

### Step 4

Select "VPC link for REST APIs", name your VPC link (e.g. "cortex"), select the API load balancer (identified in Step 1), and click "Create"

![step 3](https://user-images.githubusercontent.com/808475/80143027-03c84580-8561-11ea-92de-9ed0a5dfa593.png)

### Step 5

Wait for the VPC link to be created (it will take a few minutes)

![step 4](https://user-images.githubusercontent.com/808475/80144088-bbaa2280-8562-11ea-901b-8520eb253df7.png)

### Step 6

Go to the [API Gateway console](https://console.aws.amazon.com/apigateway/home), select "REST API" under "Choose an API type", and click "Build"

![step 5](https://user-images.githubusercontent.com/808475/78293216-18269e80-74dd-11ea-9e68-86922c2cbc7c.png)

### Step 7

Select "REST" and "New API", name your API (e.g. "cortex"), select either "Regional" or "Edge optimized" (depending on your preference), and click "Create API"

![step 6](https://user-images.githubusercontent.com/808475/78293434-66d43880-74dd-11ea-92d6-692158171a3f.png)

### Step 8

Select "Actions" > "Create Resource"

![step 7](https://user-images.githubusercontent.com/808475/80141938-3cffb600-855f-11ea-9c1c-132ca4503b7a.png)

### Step 9

Select "Configure as proxy resource" and "Enable API Gateway CORS", and click "Create Resource"

![step 8](https://user-images.githubusercontent.com/808475/80142124-80f2bb00-855f-11ea-8e4e-9413146e0815.png)

### Step 10

Select "VPC Link", select "Use Proxy Integration", choose your newly-created VPC Link, and set "Endpoint URL" to "http://<BASE_API_ENDPOINT>/{proxy}". You can get your base API endpoint via `cortex cluster info`; make sure to prepend `http://` and append `/{proxy}`. For example, mine is: `http://a5044e34a352d44b0945adcd455c7fa3-32fa161d3e5bcbf9.elb.us-west-2.amazonaws.com/{proxy}`. Click "Save"

![step 9](https://user-images.githubusercontent.com/808475/80147407-4f322200-8568-11ea-8ef5-df5164c1375f.png)

### Step 11

Select "Actions" > "Deploy API"

![step 10](https://user-images.githubusercontent.com/808475/80147555-86083800-8568-11ea-86af-1b1e38c9d322.png)

### Step 12

Create a new stage (e.g. "dev") and click "Deploy"

![step 11](https://user-images.githubusercontent.com/808475/80147631-a7692400-8568-11ea-8a09-13dbd50b17b9.png)

### Step 13

Copy your "Invoke URL"

![step 12](https://user-images.githubusercontent.com/808475/80147716-c798e300-8568-11ea-9aef-7dd6fdf4a68a.png)

### Using your new endpoint

You may now use the "Invoke URL" in place of your APIs endpoint in your client. For example, this curl request:

```bash
curl http://a5044e34a352d44b0945adcd455c7fa3-32fa161d3e5bcbf9.elb.us-west-2.amazonaws.com/iris-classifier -X POST -H "Content-Type: application/json" -d @sample.json
```

Would become:

```bash
curl https://lrivodooqh.execute-api.us-west-2.amazonaws.com/dev/iris-classifier -X POST -H "Content-Type: application/json" -d @sample.json
```

### Cleanup

Delete the API Gateway and VPC Link before spinning down your Cortex cluster:

![delete api](https://user-images.githubusercontent.com/808475/80149163-05970680-856b-11ea-9f82-61f4061a3321.png)

![delete vpc link](https://user-images.githubusercontent.com/808475/80149204-1ba4c700-856b-11ea-83f7-9741c78b6b95.png)
