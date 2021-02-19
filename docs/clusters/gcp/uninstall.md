# Uninstall

Since you may wish to have access to your data after spinning down your cluster, Cortex's bucket, stackdriver logs, and
Prometheus volume are not automatically deleted when running `cortex cluster-gcp down`.

```bash
cortex cluster-gcp down
```

The `cortex cluster-gcp down` command doesn't wait for the cluster to spin down. You can ensure that the cluster has
spun down by checking the GKE console.

## Delete Volumes

The volumes used by Cortex's Prometheus and Grafana instances are not deleted by default, as they might contain important
information. If these volumes are not required anymore, you can delete them in the GCP console. Navigate to
the [Disks](https://console.cloud.google.com/compute/disks) page (be sure to set the appropriate project), select the
volumes, and click "Delete". The Prometheus and Grafana volumes that Cortex created have a name that starts
with `gke-<cluster name>-`, and the `kubernetes.io/created-for/pvc/name` tag starts with `prometheus-` and `grafana-`
respectively.
