# Plot in-flight requests

## Step 1

Go to the CloudWatch Metrics page and click on your cluster name:

![step 1](https://user-images.githubusercontent.com/808475/78299915-7b1f3200-74eb-11ea-8399-d9a465d2d7f1.png)

## Step 2

Click on "apiName":

![step 2](https://user-images.githubusercontent.com/808475/78299940-85d9c700-74eb-11ea-966c-c7ef97d76cd8.png)

## Step 3

Select your API, and click on the "graphed metrics" tab. At this point, there are two options (due to cloudwatch retention policies):

### Step 3a

If you are only looking at data in the last 3 hours, change the statistic to "Sum" and the period to 10 seconds (this is because each replica reports it's in-flight requests once per 10 seconds). This will plot total in-flight requests across the cluster.

![step 3a](https://user-images.githubusercontent.com/808475/78299975-968a3d00-74eb-11ea-82ec-968311e029d4.png)

### Step 3b

If you want data older than 3 hours, instead sum over 1 minute, but you will need to divide the y-axis by 6 to determine the actual number of in-flight requests (since the metrics are reported every 10 seconds). This is because cloudwatch aggregates second-granular data to minute-granular data after 3 hours.
