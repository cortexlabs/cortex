# Increase request timeout

We have plans to make the request timeout configurable per-API in the API configuration ([#824](https://github.com/cortexlabs/cortex/issues/824)), but until that's implemented, it's simple to increase the timeout cluster-wide.

## Step 1

Locate the API Load Balancer for your cluster in the AWS EC2 console (on the "Load Balancers" page). You can determine which is the APIs Load Balancer by selecting the "Tags" tab and checking the `kubernetes.io/service-name` tag (it should be `istio-system/ingressgateway-apis`).

![step 1](https://user-images.githubusercontent.com/808475/78300246-08628680-74ec-11ea-8e37-daebbcb35d9c.png)

## Step 2

Click on the "Description" tab for the Load balancer, scroll down, and click "Edit idle timeout"

![step 2](https://user-images.githubusercontent.com/808475/78300569-8d4da000-74ec-11ea-8742-5459e29e973b.png)

## Step 3

Enter your desired timeout (in seconds)
