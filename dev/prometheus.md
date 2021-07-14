# Metrics

## Updating metrics

When new metrics/labels/exporters are added to be scraped by prometheus, make sure the following list **is updated** as well to keep track of what metrics/labels are needed or not.

The following is a list of metrics that are currently in use.

#### Cortex metrics

1. cortex_in_flight_requests with the following labels:
    1. api_name
1. cortex_async_request_count with the following labels:
    1. api_name
    1. api_kind
    1. status_code
1. cortex_async_active with the following labels:
    1. api_name
    1. api_kind
1. cortex_async_queued with the following labels:
    1. api_name
    1. api_kind
1. cortex_async_in_flight with the following labels:
    1. api_name
    1. api_kind
1. cortex_async_latency_bucket with the following labels:
    1. api_name
    1. api_kind
1. cortex_batch_succeeded with the following labels:
    1. api_name
1. cortex_batch_failed with the following labels:
    1. api_name
1. cortex_time_per_batch_sum with the following labels:
    1. api_name
1. cortex_time_per_batch_count with the following labels:
    1. api_name

#### Istio metrics

1. istio_requests_total with the following labels:
    1. destination_service
    1. response_code
1. istio_request_duration_milliseconds_bucket with the following labels:
    1. destination_service
    1. le
1. istio_request_duration_milliseconds_sum with the following labels:
    1. destination_service
1. istio_request_duration_milliseconds_count with the following labels:
    1. destination_service

#### Kubelet metrics
1. container_cpu_usage_seconds_total with the following labels:
    1. pod
    1. container
    1. name
1. container_memory_working_set_bytes with the following labels:
    1. pod
    1. name
    1. container

#### Kube-state-metrics metrics

1. kube_pod_container_resource_requests with the following labels:
    1. exported_pod
    1. resource
    1. exported_container (required for not dropping the values for each container of each pod)
1. kube_pod_info with the following labels:
    1. exported_pod
1. kube_deployment_status_replicas_available with the following labels:
    1. deployment
1. kube_job_status_active with the following labels:
    1. job_name

#### DCGM metrics

1. DCGM_FI_DEV_GPU_UTIL with the following labels:
    1. exported_pod
1. DCGM_FI_DEV_FB_USED with the following labels:
    1. exported_pod
1. DCGM_FI_DEV_FB_FREE with the following labels:
    1. exported_pod

#### Node metrics

1. node_cpu_seconds_total with the following labels:
    1. job
    1. mode
    1. instance
    1. cpu
1. node_load1 with the following labels:
    1. job
    1. instance
1. node_load5 with the following labels:
    1. job
    1. instance
1. node_load15 with the following labels:
    1. job
    1. instance
1. node_exporter_build_info with the following labels:
    1. job
    1. instance
1. node_memory_MemTotal_bytes with the following labels:
    1. job
    1. instance
1. node_memory_MemFree_bytes with the following labels:
    1. job
    1. instance
1. node_memory_Buffers_bytes with the following labels:
    1. job
    1. instance
1. node_memory_Cached_bytes with the following labels:
    1. job
    1. instance
1. node_memory_MemAvailable_bytes with the following labels:
    1. job
    1. instance
1. node_disk_read_bytes_total with the following labels:
    1. job
    1. instance
    1. device
1. node_disk_written_bytes_total with the following labels:
    1. job
    1. instance
    1. device
1. node_disk_io_time_seconds_total with the following labels:
    1. job
    1. instance
    1. device
1. node_filesystem_size_bytes with the following labels:
    1. job
    1. instance
    1. fstype
    1. mountpoint
    1. device
1. node_filesystem_avail_bytes with the following labels:
    1. job
    1. instance
    1. fstype
    1. device
1. node_network_receive_bytes_total with the following labels:
    1. job
    1. instance
    1. device
1. node_network_transmit_bytes_total with the following labels:
    1. job
    1. instance
    1. device

##### Prometheus rules for the node exporter

1. instance:node_cpu_utilisation:rate1m from the following metrics:
    1. node_cpu_seconds_total with the following labels:
        1. job
        1. mode
1. instance:node_num_cpu:sum from the following metrics:
    1. node_cpu_seconds_total with the following labels:
        1. job
1. instance:node_load1_per_cpu:ratio from the following metrics:
    1. node_load1 with the following labels:
        1. job
1. instance:node_memory_utilisation:ratio from the following metrics:
    1. node_memory_MemTotal_bytes with the following labels:
        1. job
    1. node_memory_MemAvailable_bytes with the following labels:
        1. job
1. instance:node_vmstat_pgmajfault:rate1m with the following metrics:
    1. node_vmstat_pgmajfault with the following labels:
        1. job
1. instance_device:node_disk_io_time_seconds:rate1m with the following metrics:
    1. node_disk_io_time_seconds_total with the following labels:
        1. job
        1. device
1. instance_device:node_disk_io_time_weighted_seconds:rate1m with the following metrics:
    1. node_disk_io_time_weighted_seconds with the following labels:
        1. job
        1. device
1. instance:node_network_receive_bytes_excluding_lo:rate1m with the following metrics:
    1. node_network_receive_bytes_total with the following labels:
        1. job
        1. device
1. instance:node_network_transmit_bytes_excluding_lo:rate1m with the following metrics:
    1. node_network_transmit_bytes_total with the following labels:
        1. job
        1. device
1. instance:node_network_receive_drop_excluding_lo:rate1m with the following metrics:
    1. node_network_receive_drop_total with the following labels:
        1. job
        1. device
1. instance:node_network_transmit_drop_excluding_lo:rate1m with the following metrics:
    1. node_network_transmit_drop_total with the following labels:
        1. job
        1. device
1. cluster:cpu_utilization:ratio with the following metrics:
    1. instance:node_cpu_utilisation:rate1m
    1. instance:node_num_cpu:sum
1. cluster:load1:ratio with the following metrics:
    1. instance:node_load1_per_cpu:ratio
1. cluster:memory_utilization:ratio with the following metrics:
    1. instance:node_memory_utilisation:ratio
1. cluster:vmstat_pgmajfault:rate1m with the following metrics:
    1. instance:node_vmstat_pgmajfault:rate1m
1. cluster:network_receive_bytes_excluding_low:rate1m with the following metrics:
    1. instance:node_network_receive_bytes_excluding_lo:rate1m
1. cluster:network_transmit_bytes_excluding_lo:rate1m with the following metrics:
    1. instance:node_network_transmit_bytes_excluding_lo:rate1m
1. cluster:network_receive_drop_excluding_lo:rate1m with the following metrics:
    1. instance:node_network_receive_drop_excluding_lo:rate1m
1. cluster:network_transmit_drop_excluding_lo:rate1m with the following metrics:
    1. instance:node_network_transmit_drop_excluding_lo:rate1m
1. cluster:disk_io_utilization:ratio with the following metrics:
    1. instance_device:node_disk_io_time_seconds:rate1m
1. cluster:disk_io_saturation:ratio with the following metrics:
    1. instance_device:node_disk_io_time_weighted_seconds:rate1m
1. cluster:disk_space_utilization:ratio with the following metrics:
    1. node_filesystem_size_bytes with the following labels:
        1. job
        1. fstype
        1. mountpoint
    1. node_filesystem_avail_bytes with the following labels:
        1. job
        1. fstype
        1. mountpoint

## Re-introducing dropped metrics/labels

If you need to add some metrics/labels back for some particular use case, comment out every `metricRelabelings:` section (except the one from the `prometheus-operator.yaml` file), determine which metrics/labels you want to add back (i.e. by using the explorer from Grafana) and then re-edit the appropriate `metricRelabelings:` sections to account for the un-dropped metrics/labels.

## Prometheus Analysis

### Go Pprof

To analyse the memory allocations of prometheus, run `kubectl port-forward prometheus-prometheus-0 9090:9090`, and then run `go tool pprof -symbolize=remote -inuse_space localhost:9090/debug/pprof/heap`. Once you get the interpreter, you can run `top` or `dot` for a more detailed hierarchy of the memory usage.

### TSDB

To analyse the TSDB of prometheus, exec into the `prometheus-prometheus-0` pod, `cd` into `/tmp`, and run the following code-block:

```bash
wget https://github.com/prometheus/prometheus/releases/download/v1.7.3/prometheus-1.7.3.linux-amd64.tar.gz
tar -xzf prometheus-*
cd prometheus-*
./tsdb analyze /prometheus | less
```

*Useful link: https://www.robustperception.io/using-tsdb-analyze-to-investigate-churn-and-cardinality*

Or you can go to `localhost:9090` -> `Status` -> `TSDB Status`, but it's not as complete as running a binary analysis.
