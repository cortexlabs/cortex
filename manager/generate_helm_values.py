import os
import yaml

def merge_override(a, b):
    "merges b into a"
    for key in b:
        if key in a:
            if isinstance(a[key], dict) and isinstance(b[key], dict):
                merge_override(a[key], b[key])
            else:
                a[key] = b[key]
        else:
            a[key] = b[key]
    return a

def main():
    cluster_config_file = os.environ["CORTEX_CLUSTER_CONFIG_FILE"]
    with open(cluster_config_file, "r") as f:
        cc = yaml.safe_load(f)
    
    values_config = {
        "cortex": {
            "cluster_name": cc["cluster_name"],
            "bucket": cc["bucket"],
            "telemetry": cc["telemetry"],
            "is_managed": cc["is_managed"],
            "namespace": cc["namespace"],
            "istio_namespace": cc["istio_namespace"],
            "image_operator": cc["image_operator"],
            "image_manager": cc["image_manager"],
            "image_downloader": cc["image_downloader"],
            "image_request_monitor": cc["image_request_monitor"],
            "image_cluster_autoscaler": cc["image_cluster_autoscaler"],
            "image_metrics_server": cc["image_metrics_server"],
            "image_fluent_bit": cc["image_fluent_bit"],
            "image_istio_proxy": cc["image_istio_proxy"],
            "image_istio_pilot": cc["image_istio_pilot"],
            "image_prometheus": cc["image_prometheus"],
            "image_prometheus_statsd_exporter": cc["image_prometheus_statsd_exporter"],
            "image_prometheus_stackdriver_sidecar": cc["image_prometheus_stackdriver_sidecar"],
            "image_prometheus_config_reloader": cc["image_prometheus_config_reloader"],
            "image_prometheus_operator": cc["image_prometheus_operator"],
        },
        "global": {
            "provider": cc["provider"],

        }
    }

    if cc["provider"] == "aws":
        values_config["cortex"] = merge_override(values_config["cortex"], {
            "image_inferentia": cc["image_inferentia"],
            "image_neuron_rtd": cc["image_neuron_rtd"],
            "image_nvidia": cc["image_nvidia"],
            "image_prometheus_to_cloudwatch": cc["image_prometheus_to_cloudwatch"],
        })

    if cc["provider"] == "gcp":
        values_config["cortex"] = merge_override(values_config["cortex"], {
            "image_google_pause": cc["image_google_pause"],
        })

    print(yaml.dump(values_config))


if __name__ == "__main__":
    main()
