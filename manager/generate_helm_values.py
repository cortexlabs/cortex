# Copyright 2021 Cortex Labs, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

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
            "instance_type": cc["instance_type"],
            "image_operator": cc["image_operator"],
            "image_manager": cc["image_manager"],
            "image_downloader": cc["image_downloader"],
            "image_cluster_autoscaler": cc["image_cluster_autoscaler"],
            "image_fluent_bit": cc["image_fluent_bit"],
            "image_istio_proxy": cc["image_istio_proxy"],
            "image_istio_pilot": cc["image_istio_pilot"],
            "image_metrics_server": cc["image_metrics_server"],
            "image_prometheus": cc["image_prometheus"],
            "image_prometheus_statsd_exporter": cc["image_prometheus_statsd_exporter"],
            "image_prometheus_config_reloader": cc["image_prometheus_config_reloader"],
            "image_prometheus_operator": cc["image_prometheus_operator"],
            "image_grafana": cc["image_grafana"],
        },
        "global": {
            "provider": cc["provider"],
        },
        "addons": {
            "kubeMetricsServer": {
                "enabled": True,
            }
        },
    }

    if cc["provider"] == "aws":
        values_config["cortex"] = merge_override(
            values_config["cortex"],
            {
                "region": cc["region"],
                "image_request_monitor": cc["image_request_monitor"],
                "image_inferentia": cc["image_inferentia"],
                "image_neuron_rtd": cc["image_neuron_rtd"],
                "image_nvidia": cc["image_nvidia"],
            },
        )
        values_config["networking"] = {
            "api-ingress": {
                "gateways": {
                    "istio-ingressgateway": {
                        "serviceAnnotations": {
                            "service.beta.kubernetes.io/aws-load-balancer-type": "nlb",
                            "service.beta.kubernetes.io/aws-load-balancer-cross-zone-load-balancing-enabled": "true",
                            "service.beta.kubernetes.io/aws-load-balancer-backend-protocol": "tcp",
                            "service.beta.kubernetes.io/aws-load-balancer-ssl-ports": "https",  # "https" is the name of the https port below"
                            "service.beta.kubernetes.io/aws-load-balancer-additional-resource-tags": "cortex.dev/load-balancer=api",
                        }
                    }
                }
            },
            "operator-ingress": {
                "gateways": {
                    "istio-ingressgateway": {
                        "serviceAnnotations": {
                            "service.beta.kubernetes.io/aws-load-balancer-type": "nlb",
                            "service.beta.kubernetes.io/aws-load-balancer-cross-zone-load-balancing-enabled": "true",
                            "service.beta.kubernetes.io/aws-load-balancer-backend-protocol": "tcp",
                            "service.beta.kubernetes.io/aws-load-balancer-ssl-ports": "https",  # "https" is the name of the https port below"
                            "service.beta.kubernetes.io/aws-load-balancer-additional-resource-tags": "cortex.dev/load-balancer=operator",
                        }
                    }
                }
            },
        }

        if cc.get("api_load_balancer_scheme") == "internal":
            values_config["networking"]["api-ingress"]["gateways"]["istio-ingressgateway"][
                "serviceAnnotations"
            ]["service.beta.kubernetes.io/aws-load-balancer-internal"] = "true"
            values_config["networking"]["operator-ingress"]["gateways"]["istio-ingressgateway"][
                "serviceAnnotations"
            ]["service.beta.kubernetes.io/aws-load-balancer-internal"] = "true"

        if cc.get("ssl_certificate_arn", "") != "":
            values_config["networking"]["api-ingress"]["gateways"]["istio-ingressgateway"][
                "serviceAnnotations"
            ]["service.beta.kubernetes.io/aws-load-balancer-ssl-cert"] = cc["ssl_certificate_arn"]

    if cc["provider"] == "gcp":
        values_config["cortex"] = merge_override(
            values_config["cortex"],
            {
                "project": cc["project"],
                "zone": cc["zone"],
                "image_google_pause": cc["image_google_pause"],
            },
        )

        if cc.get("api_load_balancer_scheme") == "internal":
            values_config["networking"] = {
                "api-ingress": {
                    "gateways": {
                        "istio-ingressgateway": {
                            "serviceAnnotations": {
                                "cloud.google.com/load-balancer-type": "Internal"
                            }
                        }
                    }
                },
                "operator-ingress": {
                    "gateways": {
                        "istio-ingressgateway": {
                            "serviceAnnotations": {
                                "cloud.google.com/load-balancer-type": "Internal"
                            }
                        }
                    }
                },
            }

    print(yaml.dump(values_config))


if __name__ == "__main__":
    main()
