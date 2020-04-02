# Plot API request time

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

The avg request latency shown in `cortex get` is averaged over the lifetime of the latest version (with a cap of 2 weeks). You can find a plot of these values in CloudWatch Metrics by following these steps:

## Step 1

Go to the CloudWatch Metrics page and click on your cluster name

![step 1](https://user-images.githubusercontent.com/808475/78299915-7b1f3200-74eb-11ea-8399-d9a465d2d7f1.png)

## Step 2

Click on "APIID, APIName, metric_type"

![step 2](https://user-images.githubusercontent.com/808475/78301024-562bbe80-74ed-11ea-97f0-ea7d99a24c7f.png)

## Step 3

Filter by API name and select all API IDs

![step 3](https://user-images.githubusercontent.com/808475/78301431-e833c700-74ed-11ea-9daa-67581374514b.png)
