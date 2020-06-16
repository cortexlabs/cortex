# Set up VPC peering

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

If you are using an internal operator load balancer (i.e. you set `operator_load_balancer_scheme: internal` in your cluster configuration file before creating your cluster), you can use VPC Peering to enable your Cortex CLI to connect to your cluster operator from another VPC so that you may run `cortex` commands. Note that because the operator validates that the CLI user is an active IAM user in the same AWS account as the Cortex cluster, it is usually unnecessary to configure the operator's load balancer to be internal.

If you are using an internal API load balancer (i.e. you set `api_load_balancer_scheme: internal` in your cluster configuration file before creating your cluster) and you disabled API Gateway for your API (i.e. you set `api_gateway: none` in the `networking` field of your api configuration), you can use VPC Peering to enable prediction requests from another VPC.

This guide illustrates how to create a VPC Peering connection between a VPC of your choice and the Cortex load balancers.

## Step 1

Navigate to AWS's EC2 Load Balancer dashboard and locate the Cortex operator's load balancer. You can determine which is the operator load balancer by inspecting the `kubernetes.io/service-name` tag:

![step 1a](https://user-images.githubusercontent.com/808475/80126132-804e2a80-8547-11ea-8ce4-57d3fd96e2c4.png)

Click back to the "Description" tab and note the VPC ID of the load balancer and the ID of each of the subnets associated with the load balancer:

![step 1b](https://user-images.githubusercontent.com/808475/80127144-c2c43700-8548-11ea-95b4-ce9d1df024cc.png)

Navigate to AWS's VPC dashboard and identify the ID and CIDR block of Cortex's VPC:

![step 1c](https://user-images.githubusercontent.com/808475/80125554-af17d100-8546-11ea-96ec-00e2aaee7100.png)

The VPC ID here should match that of the load balancer.

## Step 2

Identify the ID and CIDR block of the VPC from which you'd like to connect to the Cortex VPC.

In my case, I have a VPC in the same AWS account and region, and I can locate its ID and CIDR block from AWS's VPC dashboard:

![step 2](https://user-images.githubusercontent.com/808475/80125729-eb4b3180-8546-11ea-8d20-6bc2478747ae.png)

## Step 3

From AWS's VPC dashboard, navigate to the "Peering Connections" page, and click "Create Peering Connection":

![step 3a](https://user-images.githubusercontent.com/808475/80127600-67df0f80-8549-11ea-9e10-765a6e273b54.png)

Name your new VPC Peering Connection (I used "cortex-operator", but "cortex" or "cortex-api" may make more sense depending on your use case). Then configure the connection such that the "Requester" is the VPC from which you'll connect to the Cortex VPC, and the "Accepter" is Cortex's VPC (from step 1).

![step 3b](https://user-images.githubusercontent.com/808475/80131545-3f5a1400-854f-11ea-9ca0-c51433d3fa3d.png)

Click "Create Peering Connection", navigate back to the Peering Connections dashboard, select the newly created peering connection, and click "Actions" > "Accept Request":

![step 3c](https://user-images.githubusercontent.com/808475/80132168-21d97a00-8550-11ea-8c22-79c65710d369.png)

![step 3d](https://user-images.githubusercontent.com/808475/80132179-26059780-8550-11ea-80fc-6670fcab7026.png)

## Step 4

Navigate to the VPC Route Tables page. Select the route table for the VPC from which you'd like to connect to the Cortex cluster (in my case, I just have one route table for this VPC). Select the "Routes" tab, and click "Edit routes":

![step 4a](https://user-images.githubusercontent.com/808475/80135180-b940cc00-8554-11ea-8162-c7409090897b.png)

Add a route where the "Destination" is the CIDR block for Cortex's VPC (identified in Step 1), and the "Target" is the newly-created Peering Connection:

![step 4b](https://user-images.githubusercontent.com/808475/80137033-78968200-8557-11ea-9d84-9221b772f0fc.png)

Do not create new route tables or change subnet associations.

## Step 5

Navigate back to the VPC Route Tables page. There will be a route table for each of the subnets associated with the Cortex operator load balancer (identified in Step 1):

![step 5a](https://user-images.githubusercontent.com/808475/80138244-5dc50d00-8559-11ea-9248-fc201d011530.png)

For each of these route tables, click "Edit routes" and add a new route where the "Destination" is the CIDR block for the VPC from which you will be connecting to the Cortex cluster (identified in Step 2), and the "Target" is the newly-created Peering Connection:

![step 5b](https://user-images.githubusercontent.com/808475/80138653-f78cba00-8559-11ea-8444-406e218c3bab.png)

Repeat adding this route for each route table associated with the Cortex operator's subnets; in my case there were three. Do not create new route tables or change subnet associations.

You should now be able to use the Cortex CLI and make prediction requests from your VPC.

## Cleanup

Delete the VPC Peering connection before spinning down your Cortex cluster:

![cleanup](https://user-images.githubusercontent.com/808475/80138851-57836080-855a-11ea-92f1-06d501932a41.png)
