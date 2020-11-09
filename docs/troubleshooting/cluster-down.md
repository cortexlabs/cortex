# Cluster down failures

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

On rare occasions, `cortex cluster down` may not be able to spin down your Cortex cluster. When this happens, follow these steps:

1. If you've manually created any AWS networking resources that are pointed to the cluster or its VPC (e.g. API Gateway VPC links, custom domains, etc), delete them from the AWS console.

1. Replace "<region>" and "<cluster_name>" in the following URL, and open it in your browser: `https://console.aws.amazon.com/cloudformation/home?region=<region>#/stacks?filteringText=eksctl-<cluster_name>-`

    ![image](https://user-images.githubusercontent.com/808475/97790394-963b4880-1b85-11eb-8e27-ba5a551606b3.png)

1. For each CloudFormation stack which contains the word "nodegroup", select the stack and click "Delete".

1. Select the final stack (the one that ends in "-cluster") and click "Delete".

    If deleting the stack fails, navigate to the EC2 dashboard in the AWS console, delete the load balancers that are associated with the cluster, and try again (you can determine which load balancers are associated with the cluster by setting the correct region in the console and checking the `cortex.dev/cluster-name` tag on all load balancers). If the problem still persists, delete any other AWS resources that are blocking the stack deletion and try again. Feel free to reach out to us on [gitter](https://gitter.im/cortexlabs/cortex) if you still aren't able to delete the stacks.

1. In rare cases, you may need to delete other AWS resources associated with your Cortex cluster. For each the following resources, go to the appropriate AWS Dashboard (in the region that your cluster was in), and confirm that there are no resources left behind by the cluster: API Gateway API, API Gateway VPC Link, CloudWatch Dashboard, SQS Queues, S3 Bucket, and CloudWatch LogGroups (the Cortex bucket and log groups are not deleted by `cluster down` in order to preserve your data).
