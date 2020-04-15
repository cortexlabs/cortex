# Plot API request time

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

The `avg request` metric shown in `cortex get` corresponds to how long the average inference takes (not including time spent in the queue), and is averaged over the lifetime of the latest version of your API (with a cap of 2 weeks). You plot these values in CloudWatch Metrics by following these steps:

## Step 1

Go to the CloudWatch Metrics page and click on your cluster name (in this case my cluster is named "cortex-dev", but the default is just "cortex"):

![step 1](https://user-images.githubusercontent.com/808475/79396974-e0cedd80-7f31-11ea-85ca-f92bd6c0a175.png)

## Step 2

Click on "APIID, APIName, metric_type":

![step 2](https://user-images.githubusercontent.com/808475/79397109-2b505a00-7f32-11ea-8b78-225b13476002.png)

## Step 3

Filter by API name and select all API IDs:

![step 3](https://user-images.githubusercontent.com/808475/79397298-a154c100-7f32-11ea-88d1-98c7b492d92a.png)

## Step 4

Go to the "Graphed metrics" tab and select your desired statistic, period, and time window. The graph below shows the p99 request time (in milliseconds) calculated every 10 seconds and plotted over the last 15 minutes:

![step 4](https://user-images.githubusercontent.com/808475/79397376-cf3a0580-7f32-11ea-843a-963a489c1d5e.png)
