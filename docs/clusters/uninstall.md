# Uninstall

```bash
cortex cluster down
```

## Delete metadata and log groups

Since you may wish to have access to your data after spinning down your cluster, Cortex's bucket, log groups, and
Prometheus volume are not automatically deleted when running `cortex cluster down`.

To delete them:

```bash
# set AWS credentials
export AWS_ACCESS_KEY_ID=***
export AWS_SECRET_ACCESS_KEY=***

# identify the name of your cortex S3 bucket
aws s3 ls

# delete the S3 bucket
aws s3 rb --force s3://<bucket>

# delete the log group (replace <cluster_name> with the name of your cluster, default: cortex)
aws logs describe-log-groups --log-group-name-prefix=<cluster_name> --query logGroups[*].[logGroupName] --output text | xargs -I {} aws logs delete-log-group --log-group-name {}
```

## Delete Certificates

If you've configured a custom domain for your APIs, you can remove the SSL Certificate and Hosted Zone for the domain by
following these [instructions](networking/custom-domain.md#cleanup).

## Keep Cortex Volumes

The volumes used by Cortex's Prometheus and Grafana instances are deleted by default on a cluster down operation.
If you want to keep the metrics and dashboards volumes for any reason,
you can pass the `--keep-volumes` flag to the `cortex cluster down` command.

## Troubleshooting

On rare occasions, `cortex cluster down` may not be able to spin down your Cortex cluster. When this happens, follow
these steps:

1. If you've manually created any AWS networking resources that are pointed to the cluster or its VPC (e.g. API Gateway
   VPC links, custom domains, etc), delete them from the AWS console.

1. Replace "<region>" and "<cluster_name>" in the following URL, and open it in your
   browser: `https://console.aws.amazon.com/cloudformation/home?region=<region>#/stacks?filteringText=eksctl-<cluster_name>-`

   ![image](https://user-images.githubusercontent.com/808475/97790394-963b4880-1b85-11eb-8e27-ba5a551606b3.png)

1. For each CloudFormation stack which contains the word "nodegroup", select the stack and click "Delete".

1. Select the final stack (the one that ends in "-cluster") and click "Delete".

   If deleting the stack fails, navigate to the EC2 dashboard in the AWS console, delete the load balancers that are
   associated with the cluster, and try again (you can determine which load balancers are associated with the cluster by
   setting the correct region in the console and checking the `cortex.dev/cluster-name` tag on all load balancers). If
   the problem still persists, delete any other AWS resources that are blocking the stack deletion and try again.

1. In rare cases, you may need to delete other AWS resources associated with your Cortex cluster. For each the following
   resources, go to the appropriate AWS Dashboard (in the region that your cluster was in), and confirm that there are
   no resources left behind by the cluster: CloudWatch Dashboard, SQS Queues, S3 Bucket, and CloudWatch LogGroups (the
   Cortex bucket and log groups are not deleted by `cluster down` in order to preserve your data).
