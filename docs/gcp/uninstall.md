# Uninstall

## Spin down Cortex

```bash
# spin down Cortex
cortex cluster-gcp down

# uninstall the CLI
pip uninstall cortex
rm -rf ~/.cortex
```

If you modified your bash profile, you may wish to remove `source <(cortex completion bash)` from it (or remove `source <(cortex completion zsh)` for `zsh`).

*Note: The `cortex cluster-gcp down` command doesn't wait for the cluster to come down. You can ensure that the cluster has been removed by checking the GKE console.*
