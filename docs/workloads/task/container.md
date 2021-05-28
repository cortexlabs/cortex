# Container Implementation

## Job specification

If you need access to any parameters in the job submission (e.g. `config`), the entire job specification is available at `/cortex/spec/job.json` in your API containers' filesystems.

## Multiple containers

Your Task's pod can contain multiple containers. The `/mnt` directory is mounted to each container's filesystem, and is shared across all containers.

## Using the Cortex CLI or client

It is possible to use the Cortex CLI or client to interact with your cluster's APIs from within your API containers. All containers will have a CLI configuration file present at `/cortex/client/cli.yaml`, which is configured to connect to the cluster. In addition, the `CORTEX_CLI_CONFIG_DIR` environment variable is set to `/cortex/client` by default. Therefore, no additional configuration is required to use the CLI or Python client (which can be instantiated via `cortex.client()`).

Note: your Cortex CLI or client must match the version of your cluster (available in the `CORTEX_VERSION` environment variable).
