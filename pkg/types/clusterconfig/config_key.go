/*
Copyright 2021 Cortex Labs, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package clusterconfig

const (
	ProviderKey                            = "provider"
	InstanceTypeKey                        = "instance_type"
	AcceleratorTypeKey                     = "accelerator_type"
	AcceleratorsPerInstanceKey             = "accelerators_per_instance"
	NetworkKey                             = "network"
	SubnetKey                              = "subnet"
	MinInstancesKey                        = "min_instances"
	MaxInstancesKey                        = "max_instances"
	TagsKey                                = "tags"
	InstanceVolumeSizeKey                  = "instance_volume_size"
	InstanceVolumeTypeKey                  = "instance_volume_type"
	InstanceVolumeIOPSKey                  = "instance_volume_iops"
	SpotKey                                = "spot"
	SpotConfigKey                          = "spot_config"
	PreemptibleKey                         = "preemptible"
	InstanceDistributionKey                = "instance_distribution"
	OnDemandBaseCapacityKey                = "on_demand_base_capacity"
	OnDemandPercentageAboveBaseCapacityKey = "on_demand_percentage_above_base_capacity"
	MaxPriceKey                            = "max_price"
	InstancePoolsKey                       = "instance_pools"
	OnDemandBackupKey                      = "on_demand_backup"
	ClusterNameKey                         = "cluster_name"
	RegionKey                              = "region"
	ZoneKey                                = "zone"
	ProjectKey                             = "project"
	AvailabilityZonesKey                   = "availability_zones"
	SubnetsKey                             = "subnets"
	AvailabilityZoneKey                    = "availability_zone"
	SubnetIDKey                            = "subnet_id"
	SSLCertificateARNKey                   = "ssl_certificate_arn"
	CortexPolicyARNKey                     = "cortex_policy_arn"
	IAMPolicyARNsKey                       = "iam_policy_arns"
	BucketKey                              = "bucket"
	SubnetVisibilityKey                    = "subnet_visibility"
	NATGatewayKey                          = "nat_gateway"
	APILoadBalancerSchemeKey               = "api_load_balancer_scheme"
	OperatorLoadBalancerSchemeKey          = "operator_load_balancer_scheme"
	VPCCIDRKey                             = "vpc_cidr"
	TelemetryKey                           = "telemetry"
	ImageOperatorKey                       = "image_operator"
	ImageManagerKey                        = "image_manager"
	ImageDownloaderKey                     = "image_downloader"
	ImageRequestMonitorKey                 = "image_request_monitor"
	ImageClusterAutoscalerKey              = "image_cluster_autoscaler"
	ImageMetricsServerKey                  = "image_metrics_server"
	ImageInferentiaKey                     = "image_inferentia"
	ImageNeuronRTDKey                      = "image_neuron_rtd"
	ImageNvidiaKey                         = "image_nvidia"
	ImageFluentBitKey                      = "image_fluent_bit"
	ImageStatsdKey                         = "image_statsd"
	ImageIstioProxyKey                     = "image_istio_proxy"
	ImageIstioPilotKey                     = "image_istio_pilot"
	ImageGooglePauseKey                    = "image_google_pause"
	ImagePrometheusKey                     = "image_prometheus"
	ImagePrometheusConfigReloaderKey       = "image_prometheus_config_reloader"
	ImagePrometheusOperatorKey             = "image_prometheus_operator"
	ImagePrometheusStatsDExporterKey       = "image_prometheus_statsd_exporter"
	ImagePrometheusNodeExporterKey         = "image_prometheus_node_exporter"
	ImageKubeRBACProxyKey                  = "image_kube_rbac_proxy"
	ImageGrafanaKey                        = "image_grafana"

	// User facing string
	ProviderUserKey                            = "provider"
	APIVersionUserKey                          = "cluster version"
	ClusterNameUserKey                         = "cluster name"
	RegionUserKey                              = "aws region"
	ZoneUserKey                                = "gcp zone"
	ProjectUserKey                             = "gcp project"
	AvailabilityZonesUserKey                   = "availability zones"
	SubnetsUserKey                             = "subnets"
	AvailabilityZoneUserKey                    = "availability zone"
	SubnetIDUserKey                            = "subnet id"
	SSLCertificateARNUserKey                   = "ssl certificate arn"
	CortexPolicyARNUserKey                     = "cortex policy arn"
	IAMPolicyARNsUserKey                       = "iam policy arns"
	BucketUserKey                              = "s3 bucket"
	SpotUserKey                                = "use spot instances"
	PreemptibleUserKey                         = "use preemptible instances"
	InstanceTypeUserKey                        = "instance type"
	AcceleratorTypeUserKey                     = "accelerator type"
	AcceleratorsPerInstanceUserKey             = "accelerators per instance"
	NetworkUserKey                             = "network"
	SubnetUserKey                              = "subnet"
	MinInstancesUserKey                        = "min instances"
	MaxInstancesUserKey                        = "max instances"
	TagsUserKey                                = "tags"
	InstanceVolumeSizeUserKey                  = "instance volume size (Gi)"
	InstanceVolumeTypeUserKey                  = "instance volume type"
	InstanceVolumeIOPSUserKey                  = "instance volume iops"
	InstanceDistributionUserKey                = "spot instance distribution"
	OnDemandBaseCapacityUserKey                = "spot on demand base capacity"
	OnDemandPercentageAboveBaseCapacityUserKey = "spot on demand percentage above base capacity"
	MaxPriceUserKey                            = "spot max price ($ per hour)"
	InstancePoolsUserKey                       = "spot instance pools"
	OnDemandBackupUserKey                      = "on demand backup"
	SubnetVisibilityUserKey                    = "subnet visibility"
	NATGatewayUserKey                          = "nat gateway"
	APILoadBalancerSchemeUserKey               = "api load balancer scheme"
	OperatorLoadBalancerSchemeUserKey          = "operator load balancer scheme"
	VPCCIDRUserKey                             = "vpc cidr"
	TelemetryUserKey                           = "telemetry"
	ImageOperatorUserKey                       = "operator image"
	ImageManagerUserKey                        = "manager image"
	ImageDownloaderUserKey                     = "downloader image"
	ImageRequestMonitorUserKey                 = "request monitor image"
	ImageClusterAutoscalerUserKey              = "cluster autoscaler image"
	ImageMetricsServerUserKey                  = "metrics server image"
	ImageInferentiaUserKey                     = "inferentia image"
	ImageNeuronRTDUserKey                      = "neuron rtd image"
	ImageNvidiaUserKey                         = "nvidia image"
	ImageFluentBitUserKey                      = "fluent-bit image"
	ImageIstioProxyUserKey                     = "istio proxy image"
	ImageIstioPilotUserKey                     = "istio pilot image"
	ImageGooglePauseUserKey                    = "google pause image"
	ImagePrometheusUserKey                     = "prometheus image"
	ImagePrometheusConfigReloaderUserKey       = "prometheus config reloader image"
	ImagePrometheusOperatorUserKey             = "prometheus operator image"
	ImagePrometheusStatsDExporterUserKey       = "prometheus statsd exporter image"
	ImagePrometheusNodeExporterUserKey         = "prometheus node exporter image"
	ImageKubeRBACProxyUserKey                  = "kube rbac proxy image"
	ImageGrafanaUserKey                        = "grafana image"
)
