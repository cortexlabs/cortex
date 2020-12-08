# Uninstall

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

## Spin down Cortex

```bash
# spin down Cortex
cortex cluster down

# uninstall the CLI
pip uninstall cortex
rm -rf ~/.cortex
```

If you modified your bash profile, you may wish to remove `source <(cortex completion bash)` from it (or remove `source <(cortex completion zsh)` for `zsh`).

*Note: The `cortex cluster down` command doesn't wait for the cluster to come down. You should ensure that the cluster has been removed by checking your GCP console.*
