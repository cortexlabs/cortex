# API is stuck updating

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

If your API is stuck in the "updating" or "compute unavailable" state (which is displayed when running `cortex get`), there are a few possible causes. Here are some things to check:

## Check `cortex logs API_NAME`

If no logs appear (e.g. it just says "fetching logs..."), continue down this list.

## Check `max_instances` for your cluster

When you created your Cortex cluster, you configured `max_instances` (either from the command prompts or via a cluster configuration file, e.g. `cluster.yaml`). If your cluster already has `min_instances` running instances, additional instances cannot be created and APIs may not be able to deploy, scale, or update.

You can check the current value of `max_instances` by running `cortex cluster info` (or `cortex cluster info --config cluster.yaml` if you have a cluster configuration file).

You can update `max_instances` by running `cortex cluster configure` (or by modifying `max_instances` in your cluster configuration file and running `cortex cluster configure --config cluster.yaml`).

## Check your AWS auto scaling group activity history

In most cases when AWS is unable to provision additional instances, the reason will be logged in the auto scaling group's activity history.

Here is how you can check these logs:

1. Log in to the AWS console and go to the EC2 service page
2. Click "Auto Scaling Groups" on the bottom of the side panel on the left
3. Select one of the "worker" autoscaling groups for your cluster (there may be two)
4. Click the "Activity" tab at the bottom half of the screen (it may also be called "Activity History" depending on which AWS console UI you're using)
5. Scroll down (if necessary) and inspect the activity history, looking for any errors and their causes
6. Repeat steps 3-5 for the other worker autoscaling group (if applicable)

Here is how it looks on the new console UI:

![new ui](https://user-images.githubusercontent.com/808475/78153371-852d2c00-742a-11ea-9bde-dbad5c603f8f.png)

On the old UI:

![old ui](https://user-images.githubusercontent.com/808475/78153350-7e9eb480-742a-11ea-9221-1f6559db45fd.png)

The most common reason AWS is unable to provision instances is that you have reached your instance limit:

* **on-demand instances**: You may have limited access to your requested instance type. To check your limits, click [here](https://console.aws.amazon.com/ec2/v2/home?#Limits:), set your region in the upper right, and type "on-demand" in the search box. You can request a limit by selecting your instance family and clicking "Request limit increase" in the upper right. Note that the limits are vCPU-based no matter the instance type (e.g. to run 4 `g4dn.xlarge` instances, you will need a 16 vCPU limit for G instances).

* **spot instances**: You may have limited access to spot instances in your region. To check your limits, click [here](https://console.aws.amazon.com/ec2/v2/home?#Limits:), set your region in the upper right, and type "spot" in the search box. Note that the listed spot instance limit may misrepresent the actual number of spot instances you can allocate. Your actual spot instance limit depends on the instance type you have requested. In general, you can run a higher number of smaller instance types, or fewer large instance types. For example, even if the limit shows `20`, if you are requesting large instances like `p2.xlarge`, the actual limit may be lower due to the way AWS calculates this limit. If you are not getting the number of spot instances that you are expecting for your instance type, you can request a limit increase [here](https://console.aws.amazon.com/support/home#/case/create?issueType=service-limit-increase&limitType=service-code-ec2-spot-instances).

If you are using spot instances and don't have `on_demand_backup` set to true, it is also possible that AWS has run out of spot instances for your requested instance type and region. You can enable `on_demand_backup` to allow Cortex to fall back to on-demand instances when spot instances are unavailable, or you can try adding additional alternative instance types in `instance_distribution`. See our [spot documentation](../cluster-management/spot-instances.md).

## Disabling rolling updates

By default, cortex performs rolling updates on all APIs. This is to ensure that traffic can continue to be served during updates, and that there is no downtime if there's an error in the new version. However, this can lead to APIs getting stuck in the "updating" state when the cluster is unable to increase its instance count (e.g. for one of the reasons above).

Here is an example: You set `max_instances` to 1, or your AWS account limits you to a single `g4dn.xlarge` instance (i.e. your G instance vCPU limit is 4). You have an API running which requested 1 GPU. When you update your API via `cortex deploy`, Cortex attempts to deploy the updated version, and will only take down the old version once the new one is running. In this case, since there is no GPU available on the single running instance (it's taken by the old version of your API), the new version of your API requests a new instance to run on. Normally this will be ok (it might just take a few minutes since a new instance has to spin up): the new instance will become live, the new API replica will run on it, once it starts up successfully the old replica will be terminated, and eventually the old instance will spin down. In this case, however, the new version gets stuck because the second instance cannot be created, and the first instance cannot be freed up until the new version is running.

If you're running in a development environment, this rolling update behavior can be undesirable.

You can disable rolling updates for your API in your API configuration (e.g. in `cortex.yaml`): set `max_surge` to 0 (in the `update_strategy` configuration). E.g.:

```yaml
- name: text-generator
  predictor:
    type: python
    ...
  update_strategy:
    max_surge: 0
```
