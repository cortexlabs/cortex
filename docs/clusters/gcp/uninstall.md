# Uninstall

Since you may wish to have access to your data after spinning down your cluster, Cortex's bucket, stackdriver logs, and
Prometheus volume are not automatically deleted when running `cortex cluster-gcp down --config cluster.yaml`.

```bash
cortex cluster-gcp down --config cluster.yaml
```

The `cortex cluster-gcp down --config cluster.yaml` command doesn't wait for the cluster to spin down. You can ensure that the cluster has
spun down by checking the GKE console.

## Keep Cortex Volumes

The volumes used by Cortex's Prometheus and Grafana instances are deleted by default on a cluster down operation.
If you want to keep the metrics and dashboards volumes for any reason,
you can pass the `--keep-volumes` flag to the `cortex cluster-gcp down --config cluster.yaml` command.
