# Set up AWS API Gateway

We have plans to automatically configure API gateway when creating a Cortex API ([#326](https://github.com/cortexlabs/cortex/issues/326)), but until that's implemented, it's fairly straightforward to set it up manually. One reason to use API Gateway is to get https working with valid certificates (either by using AWS's built-in certificates, or using your own via custom domains and the AWS Certificate Manager).

## Step 1

Go to the [API Gateway console](https://console.aws.amazon.com/apigateway/home), select "REST API" under "Choose an API type", and click "Build"

![step 1](https://user-images.githubusercontent.com/808475/78293216-18269e80-74dd-11ea-9e68-86922c2cbc7c.png)

## Step 2

Select "REST" and "New API", name your API (e.g. "cortex"), select either "Regional" or "Edge optimized" (depending on your preference), and click "Create API"

![step 2](https://user-images.githubusercontent.com/808475/78293434-66d43880-74dd-11ea-92d6-692158171a3f.png)

## Step 3

Select "Actions" > "Create Resource"

![step 3](https://user-images.githubusercontent.com/808475/78293491-7b183580-74dd-11ea-9604-54a4bf407320.png)

## Step 4

Select "Configure as proxy resource", name the resource (e.g. "Cortex API Proxy"), set the Resource Path to "{proxy+}", select "Enable API Gateway CORS", and click "Create Resource"

![step 4](https://user-images.githubusercontent.com/808475/78293551-91be8c80-74dd-11ea-9af1-673dce03c36e.png)

## Step 5

Select "HTTP Proxy" and set "Endpoint URL" to "http://<APIS_ENDPOINT>/{proxy}". You can get your API endpoint via `cortex cluster info`; make sure to prepend `http://` and append `/{proxy}`. For example: `http://ade70f94675cf40b89bdf6acb7ddc55e-1557058080.us-west-2.elb.amazonaws.com/{proxy}`

Leave "Content Handling" set to "Passthrough" and Click "Save"

![step 5](https://user-images.githubusercontent.com/808475/78293586-a69b2000-74dd-11ea-9b16-8e8f544e700a.png)

## Step 6

Select "Actions" > "Deploy API"

![step 6](https://user-images.githubusercontent.com/808475/78293643-bdda0d80-74dd-11ea-9295-8dba59785e78.png)

## Step 7

Create a new stage (e.g. "dev") and click "Deploy"

![step 7](https://user-images.githubusercontent.com/808475/78293672-d0ecdd80-74dd-11ea-93cf-21420a7bb94f.png)

## Step 8

Copy your "Invoke URL"

![step 8](https://user-images.githubusercontent.com/808475/78293715-e6620780-74dd-11ea-96dd-12a3b60037ca.png)

## Using your new endpoint

You may now use the "Invoke URL" in place of your APIs endpoint in your client. For example, this curl request:

```bash
curl http://ade70f94675cf40b89bdf6acb7ddc55e-1557058080.us-west-2.elb.amazonaws.com/iris-classifier -X POST -H "Content-Type: application/json" -d @sample.json
```

Would become:

```bash
curl https://b32zr0fca9.execute-api.us-west-2.amazonaws.com/dev/iris-classifier -X POST -H "Content-Type: application/json" -d @sample.json
```
