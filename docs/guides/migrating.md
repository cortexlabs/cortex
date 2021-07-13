# Cluster migrations

This guide is for users who want to upgrade to a new version of a Cortex cluster or migrate from one cluster to another.

Remember don't spin down your existing Cortex cluster until your new cluster is receiving traffic to mitigate down time.

## Setup a subdomain using route 53 hosted zone

If you've already setup a subdomain with route 53 hosted zone pointing to a cluster, skip this step.

Setting up a route 53 hosted zone allows you to transfer traffic seamlessly from from an existing cluster to a new cluster and minimize downtime. You can find the instructions for setting up a subdomain [here](./custom-domain.md). You will need to update clients interacting with your Cortex APIs to point to the subdomain.

## Export all of the APIs from previous cluster

The export command can be used to get a list the API yaml specifications deployed in the previous cluster.

```bash
cortex cluster export --name <previous_cluster_name> --region <region>
```

## Spin up a new cortex cluster

If you are spinning up a new cluster with the same version:

```bash
cortex cluster up new-cluster.yaml --configure-env cortex2
```

This will create a CLI environment named `cortex2` for accessing the new cluster.

Otherwise install a new version of the Cortex CLI. You will have to use a new virtual env or conda env to switch between the two versions of Cortex.

```bash
pip install cortex==<desired version>

# confirm version
cortex version

# spin up your cluster and setup a new environment named cortex2
cortex cluster up cluster.yaml --configure-env cortex2
```

## Deploy the APIs to your new cluster

Please read the documentation for the new cluster's version to look at the changes to API configuration and container implementation. You may need to make modifications to the API yaml to suit the cluster configuration and version of your new cluster.

```bash
cortex deploy -e cortex2 <api_spec_file>
```

After you've updated the API specifications and images if necessary, you can deploy them onto your new cluster.

## Point your custom domain to your new cluster

Verify that all of the APIs in your new cluster are working as expected by accessing via the cluster's API loadbalancer URL.

Get the cluster's API loadbalancer URL

```bash
cortex cluster info --name <new_cluster_name> --region <region>
```

Once the APIs on the new cluster have been verified, navigate to the A record in your custom domains's route 53 hosted zone and updated the Alias to point the new cluster's API loadbalancer URL.

Rather than suddenly routing all of your traffic from the old cluster to the new cluster, you can use [weighted records](https://docs.aws.amazon.com/Route53/latest/DeveloperGuide/routing-policy.html#routing-policy-weighted) to incrementally route more traffic to your new cluster.

## Spin down the old cluster

After confirming that your old cluster has serviced existing traffic and has no new incoming traffic, spin down the older cluster.

```bash
cortex cluster down --name <old_cluster_name> --region <region>
```
