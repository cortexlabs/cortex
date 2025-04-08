# Update

## Modify existing cluster

You can add or remove node groups, resize existing node groups, and update some configuration fields of a running cluster.

Fetch the current cluster configuration:

```bash
cortex cluster info --print-config --name CLUSTER_NAME --region REGION > cluster.yaml
```

Make your desired changes, and then apply them:

```bash
cortex cluster configure cluster.yaml
```

Cortex will calculate the difference and you will be prompted with the update plan.

If you would like to update fields that cannot be modified on a running cluster, you must create a new cluster with your desired configuration.

## Upgrade to a new version

Updating an existing Cortex cluster is not supported at the moment. Please spin down the previous version of the cluster, install the latest version of the Cortex CLI, and use it to spin up a new Cortex cluster. See the next section for how to do this without downtime.

## Update or upgrade without downtime

It is possible to update to a new version Cortex or to migrate from one cluster to another without downtime.

Note: it is important to not spin down your previous cluster until after your new cluster is receiving traffic.

### Set up a subdomain using a Route 53 hosted zone

If you've already set up a subdomain with a Route 53 hosted zone pointing to your cluster, skip this step.

Setting up a Route 53 hosted zone allows you to transfer traffic seamlessly from from an existing cluster to a new cluster, thereby avoiding downtime. You can find the instructions for setting up a subdomain [here](../networking/custom-domain.md). You will need to update any clients interacting with your Cortex APIs to point to the new subdomain.

### Export all APIs from your previous cluster

The `cluster export` command can be used to get the YAML specifications of all APIs deployed in your cluster:

```bash
cortex cluster export --name <previous_cluster_name> --region <region>
```

### Spin up a new cortex cluster

If you are creating a new cluster with the same Cortex version:

```bash
cortex cluster up new-cluster.yaml --configure-env cortex2
```

This will create a CLI environment named `cortex2` for accessing the new cluster.

If you are spinning a up a new cluster with a different Cortex version, first install the cortex CLI matching the desired cluster version:

<!-- CORTEX_VERSION_README x2 -->

```bash
# download the desired CLI version, replace 0.42.2 with the desired version (Note the "v"):
bash -c "$(curl -sS https://raw.githubusercontent.com/cortexlabs/cortex/v0.42.2/get-cli.sh)"

# confirm Cortex CLI version
cortex version

# spin up your cluster using the new CLI version
cortex cluster up cluster.yaml --configure-env cortex2
```

You can use different Cortex CLIs to interact with the different versioned clusters; here is an example:

<!-- CORTEX_VERSION_README x4 -->

```bash
# download the desired CLI version, replace 0.42.2 with the desired version (Note the "v"):
CORTEX_INSTALL_PATH=$(pwd)/cortex0.42.2 bash -c "$(curl -sS https://raw.githubusercontent.com/cortexlabs/cortex/v0.42.2/get-cli.sh)"

# confirm cortex CLI version
./cortex0.42.2 version
```

### Deploy the APIs to your new cluster

Please read the [changelogs](https://github.com/cortexlabs/cortex/releases) and the latest documentation to identify any features and breaking changes in the new version. You may need to make modifications to your cluster and/or API configuration files.

```bash
cortex deploy -e cortex2 <api_spec_file>
```

After you've updated the API specifications and images if necessary, you can deploy them onto your new cluster.

### Point your custom domain to your new cluster

Verify that all of the APIs in your new cluster are working as expected by accessing via the cluster's API load balancer URL.

Get the cluster's API load balancer URL:

```bash
cortex cluster info --name <new_cluster_name> --region <region>
```

Once the APIs on the new cluster have been verified as working properly, it is recommended to update `min_replicas` of your APIs on the new cluster to match the current values in your previous cluster. This will avoid large sudden scale-up events as traffic is shifted to the new cluster.

Then, navigate to the A record in your custom domains's Route 53 hosted zone and update the Alias to point the new cluster's API load balancer URL. Rather than suddenly routing all of your traffic from the previous cluster to the new cluster, you can use [weighted records](https://docs.aws.amazon.com/Route53/latest/DeveloperGuide/routing-policy.html#routing-policy-weighted) to incrementally route more traffic to your new cluster.

If you increased `min_replicas` for your APIs in the new cluster during the transition, you may reduce `min_replicas` back to your desired level once all traffic has been shifted.

### Spin down the previous cluster

After confirming that your previous cluster has completed servicing all existing traffic and is not receiving any new traffic, spin down your previous cluster:

```bash
# Note: it is recommended to install the Cortex CLI matching the previous cluster's version to ensure proper deletion.

cortex cluster down --name <previous_cluster_name> --region <region>
```
