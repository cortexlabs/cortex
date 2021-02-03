# Uninstall

Identify the Cortex installation's release name:

```bash
helm list
```

Uninstall Cortex:

```bash
helm uninstall <RELEASE_NAME>
```

Resources in your cloud such as the bucket and logs will not be deleted.
