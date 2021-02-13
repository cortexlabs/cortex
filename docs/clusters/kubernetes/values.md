# `values.yaml`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| global.provider | string | `""` | "aws" or "gcp" (required) |
| global.hub | string | `"quay.io/cortexlabs"` | Default hub for istio images |
| global.tag | string | `"master"` | Default tag for istio images |
| global.proxy.image | string | `"istio-proxy"` | Image for the istio proxy controller |
| cortex.cluster_name | string | `""` | Name of the cluster (required) |
| cortex.bucket | string | `""` | "my-cortex-bucket" (without s3:// or gs://) (required) |
| cortex.region | string | `""` | AWS-only (region where the cluster was provisioned) (required) |
| cortex.zone | string | `""` | GCP-only (zone where the cluster was provisioned) (required) |
| cortex.project | string | `""` | The project ID (required) |
| cortex.image_operator | string | `"quay.io/cortexlabs/operator:master"` |  |
| cortex.image_manager | string | `"quay.io/cortexlabs/manager:master"` |  |
| cortex.image_downloader | string | `"quay.io/cortexlabs/downloader:master"` |  |
| cortex.image_request_monitor | string | `"quay.io/cortexlabs/request-monitor:master"` |  |
| cortex.image_cluster_autoscaler | string | `"quay.io/cortexlabs/cluster-autoscaler:master"` |  |
| cortex.image_metrics_server | string | `"quay.io/cortexlabs/metrics-server:master"` |  |
| cortex.image_inferentia | string | `"quay.io/cortexlabs/inferentia:master"` |  |
| cortex.image_neuron_rtd | string | `"quay.io/cortexlabs/neuron-rtd:master"` |  |
| cortex.image_nvidia | string | `"quay.io/cortexlabs/nvidia:master"` |  |
| cortex.image_fluent_bit | string | `"quay.io/cortexlabs/fluent-bit:master"` |  |
| cortex.image_istio_proxy | string | `"quay.io/cortexlabs/istio-proxy:master"` |  |
| cortex.image_istio_pilot | string | `"quay.io/cortexlabs/istio-pilot:master"` |  |
| cortex.image_google_pause | string | `"quay.io/cortexlabs/pause:master"` |  |
| cortex.image_prometheus | string | `"quay.io/cortexlabs/prometheus:master"` |  |
| cortex.image_prometheus_config_reloader | string | `"quay.io/cortexlabs/prometheus-config-reloader:master"` |  |
| cortex.image_prometheus_operator | string | `"quay.io/cortexlabs/prometheus-operator:master"` |  |
| cortex.image_prometheus_statsd_exporter | string | `"quay.io/cortexlabs/prometheus-statsd-exporter:master"` |  |
| cortex.image_grafana | string | `"quay.io/cortexlabs/grafana:master"` |  |
| cortex.resources.requests.cpu | string | `"200m"` |  |
| cortex.resources.requests.memory | string | `"128Mi"` |  |
| cortex.resources.limits.cpu | string | `"2000m"` |  |
| cortex.resources.limits.memory | string | `"1024Mi"` |  |
| networking.api-ingress.gateways.istio-ingressgateway.type | string | `"LoadBalancer"` | "ClusterIP", "NodePort" or "LoadBalancer"; for API ingress (RealtimeAPI, BatchAPI, TaskAPI, TrafficSplitter) |
| networking.api-ingress.gateways.istio-ingressgateway.externalTrafficPolicy | string | `"Cluster"` | "Local" or "Cluster" |
| networking.api-ingress.gateways.istio-ingressgateway.serviceAnnotations | list | `[]` | Annotations to configure your ingress (specific to aws or gcp) |
| networking.operator-ingress.gateways.istio-ingressgateway.type | string | `"LoadBalancer"` | "ClusterIP", "NodePort" or "LoadBalancer"; for operator ingress (CLI/Python Client) |
| networking.operator-ingress.gateways.istio-ingressgateway.externalTrafficPolicy | string | `"Cluster"` | "Local" or "Cluster" |
| networking.operator-ingress.gateways.istio-ingressgateway.serviceAnnotations | list | `[]` | Annotations to configure your ingress (specific to aws or gcp) |
| networking.istio-discovery.pilot.hub | string | `"quay.io/cortexlabs"` |  |
| networking.istio-discovery.pilot.tag | string | `"master"` |  |
| networking.istio-discovery.pilot.image | string | `"istio-pilot"` |  |
| prometheus.volumeClaimSize | string | `"40Gi"` | Size of the volume for prometheus |
| prometheus.retentionSize | string | `"35GB"` | How much prometheus will store in its volume |
| prometheus.retentionWindow | string | `"2w"` | Retention time window |
| prometheus.monitorResources.requests.cpu | string | `nil` |  |
| prometheus.monitorResources.requests.memory | string | `"400Mi"` |  |
| prometheus.monitorResources.limits.cpu | string | `nil` |  |
| prometheus.monitorResources.limits.memory | string | `nil` |  |
| prometheus.operatorResources.requests.cpu | string | `"100m"` |  |
| prometheus.operatorResources.requests.memory | string | `"100Mi"` |  |
| prometheus.operatorResources.limits.cpu | string | `"200m"` |  |
| prometheus.operatorResources.limits.memory | string | `"200Mi"` |  |
| prometheusStatsDExporter.resources.requests.cpu | string | `"100m"` |  |
| prometheusStatsDExporter.resources.requests.memory | string | `"100Mi"` |  |
| prometheusStatsDExporter.resources.limits.cpu | string | `nil` |  |
| prometheusStatsDExporter.resources.limits.memory | string | `"100Mi"` |  |
| addons.grafana.enabled | bool | `true` | Whether grafana is enabled or not |
| addons.grafana.storage | string | `"2Gi"` | How much grafana can store in its volume |
| addons.grafana.resources.requests.cpu | string | `"100m"` |  |
| addons.grafana.resources.requests.memory | string | `"100Mi"` |  |
| addons.grafana.resources.limits.cpu | string | `"200m"` |  |
| addons.grafana.resources.limits.memory | string | `"200Mi"` |  |
| addons.fluentbit.enabled | bool | `true` | Whether fluentbit is enabled or not; used for exporting logs to CloudWatch or Stackdriver |
| addons.fluentbit.resources.requests.cpu | string | `"100m"` |  |
| addons.fluentbit.resources.requests.memory | string | `"150Mi"` |  |
| addons.fluentbit.resources.limits.cpu | string | `nil` |  |
| addons.fluentbit.resources.limits.memory | string | `"150Mi"` |  |
| addons.kubeMetricsServer.enabled | bool | `false` | Whether the kube metrics server is enabled or not; for "kubectl top" command |
