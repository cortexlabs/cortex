# Plot in-flight requests

## Step 1

Go to the CloudWatch Metrics page and click on your cluster name (in this case my cluster is named "cortex-dev", but the default is just "cortex"):

![step 1](https://user-images.githubusercontent.com/808475/79396974-e0cedd80-7f31-11ea-85ca-f92bd6c0a175.png)

## Step 2

Click on "apiName":

![step 2](https://user-images.githubusercontent.com/808475/79397644-67d08580-7f33-11ea-9310-f599d1289809.png)

## Step 3

Select your API:

![step 3](https://user-images.githubusercontent.com/808475/79397891-1f659780-7f34-11ea-9ea1-7ed9365b516d.png)

## Step 4

Click on the "graphed metrics" tab.

At this point, there are two options (due to cloudwatch retention policies):

### Step 4a

If you are only looking at data in the last 3 hours, change the statistic to "Sum" and the period to 10 seconds (this is because each replica reports it's in-flight requests once per 10 seconds). This will plot total in-flight requests for your selected API (across all replicas). You can change the time window for your plot in the upper right corner.

![step 4a](https://user-images.githubusercontent.com/808475/79397858-05c45000-7f34-11ea-8c44-badfae2a50e2.png)

### Step 4b

If you want data older than 3 hours, instead sum over 1 minute, but you will need to divide the y-axis by 6 to determine the actual number of in-flight requests (since the metrics are reported every 10 seconds). This is because cloudwatch aggregates second-granular data to minute-granular data after 3 hours.
