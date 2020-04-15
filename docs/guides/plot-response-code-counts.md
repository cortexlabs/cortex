# Plot response code counts

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

This plot shows the number of requests that result in 2XX, 4XX, and 5XX response codes over time.

## Step 1

Go to the CloudWatch Metrics page and click on your cluster name (in this case my cluster is named "cortex-dev", but the default is just "cortex"):

![step 1](https://user-images.githubusercontent.com/808475/79396974-e0cedd80-7f31-11ea-85ca-f92bd6c0a175.png)

## Step 2

Click on "APIID, APIName, Code, metric_type":

![step 2](https://user-images.githubusercontent.com/808475/79398268-19bc8180-7f35-11ea-99cc-43b7790d2f01.png)

## Step 3

Filter by API name and select all API IDs:

![step 3](https://user-images.githubusercontent.com/808475/79398332-4d97a700-7f35-11ea-925f-20ab0f6eff49.png)

## Step 4

Go to the "Graphed metrics" tab and select your desired statistic, period, and time window. The graph below shows the total number of requests per minute plotted over the last 15 minutes (this was during one of our load tests):

![step 4](https://user-images.githubusercontent.com/808475/79398568-d9a9ce80-7f35-11ea-859d-1f949c8e2ff8.png)
