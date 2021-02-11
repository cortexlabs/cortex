# Uninstall

```bash
cortex cluster-gcp down
```

The `cortex cluster-gcp down` command doesn't wait for the cluster to spin down. You can ensure that the cluster has spun down by checking the GKE console.

## Delete Prometheus Volume

The volume used by Cortex's Prometheus instance is not deleted by default, as it might contain important information. If this volume is not required anymore, you can delete it in the GCP console. Navigate to the [Disks](https://console.cloud.google.com/compute/disks) page (be sure to set the appropriate project), select the volume, and click "Delete". A Prometheus volume that Cortex created has a name that starts with `gke-<cluster name>-`, and the `kubernetes.io/created-for/pvc/name` tag starts with `prometheus-`.
