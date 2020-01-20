/*
Copyright 2020 Cortex Labs, Inc.

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
	InstanceTypeKey                        = "instance_type"
	MinInstancesKey                        = "min_instances"
	MaxInstancesKey                        = "max_instances"
	InstanceVolumeSizeKey                  = "instance_volume_size"
	SpotKey                                = "spot"
	SpotConfigKey                          = "spot_config"
	InstanceDistributionKey                = "instance_distribution"
	OnDemandBaseCapacityKey                = "on_demand_base_capacity"
	OnDemandPercentageAboveBaseCapacityKey = "on_demand_percentage_above_base_capacity"
	MaxPriceKey                            = "max_price"
	InstancePoolsKey                       = "instance_pools"
	OnDemandBackupKey                      = "on_demand_backup"
	ClusterNameKey                         = "cluster_name"
	RegionKey                              = "region"
	AvailabilityZonesKey                   = "availability_zones"
	BucketKey                              = "bucket"
	LogGroupKey                            = "log_group"
	TelemetryKey                           = "telemetry"
	ImagePythonServeKey                    = "image_python_serve"
	ImagePythonServeGPUKey                 = "image_python_serve_gpu"
	ImageTFServeKey                        = "image_tf_serve"
	ImageTFServeGPUKey                     = "image_tf_serve_gpu"
	ImageTFAPIKey                          = "image_tf_api"
	ImageONNXServeKey                      = "image_onnx_serve"
	ImageONNXServeGPUKey                   = "image_onnx_serve_gpu"
	ImageOperatorKey                       = "image_operator"
	ImageManagerKey                        = "image_manager"
	ImageDownloaderKey                     = "image_downloader"
	ImageClusterAutoscalerKey              = "image_cluster_autoscaler"
	ImageMetricsServerKey                  = "image_metrics_server"
	ImageNvidiaKey                         = "image_nvidia"
	ImageFluentdKey                        = "image_fluentd"
	ImageStatsdKey                         = "image_statsd"
	ImageIstioProxyKey                     = "image_istio_proxy"
	ImageIstioPilotKey                     = "image_istio_pilot"
	ImageIstioCitadelKey                   = "image_istio_citadel"
	ImageIstioGalleyKey                    = "image_istio_galley"

	// User facing string
	APIVersionUserKey                          = "cluster version"
	ClusterNameUserKey                         = "cluster name"
	RegionUserKey                              = "aws region"
	AvailabilityZonesUserKey                   = "availability zones"
	BucketUserKey                              = "s3 bucket"
	SpotUserKey                                = "use spot instances"
	InstanceTypeUserKey                        = "instance type"
	MinInstancesUserKey                        = "min instances"
	MaxInstancesUserKey                        = "max instances"
	InstanceVolumeSizeUserKey                  = "instance volume size (Gi)"
	InstanceDistributionUserKey                = "spot instance distribution"
	OnDemandBaseCapacityUserKey                = "spot on demand base capacity"
	OnDemandPercentageAboveBaseCapacityUserKey = "spot on demand percentage above base capacity"
	MaxPriceUserKey                            = "spot max price ($ per hour)"
	InstancePoolsUserKey                       = "spot instance pools"
	OnDemandBackupUserKey                      = "on demand backup"
	LogGroupUserKey                            = "cloudwatch log group"
	TelemetryUserKey                           = "telemetry"
	ImagePythonServeUserKey                    = "python serving image"
	ImagePythonServeGPUUserKey                 = "python serving gpu image"
	ImageTFServeUserKey                        = "tensorflow serving image"
	ImageTFServeGPUUserKey                     = "tensorflow serving gpu image"
	ImageTFAPIUserKey                          = "tensorflow api image"
	ImageONNXServeUserKey                      = "onnx serving image"
	ImageONNXServeGPUUserKey                   = "onnx serving gpu image"
	ImageOperatorUserKey                       = "operator image"
	ImageManagerUserKey                        = "manager image"
	ImageDownloaderUserKey                     = "downloader image"
	ImageClusterAutoscalerUserKey              = "cluster autoscaler image"
	ImageMetricsServerUserKey                  = "metrics server image"
	ImageNvidiaUserKey                         = "nvidia image"
	ImageFluentdUserKey                        = "fluentd image"
	ImageStatsdUserKey                         = "statsd image"
	ImageIstioProxyUserKey                     = "istio proxy image"
	ImageIstioPilotUserKey                     = "istio pilot image"
	ImageIstioCitadelUserKey                   = "istio citadel image"
	ImageIstioGalleyUserKey                    = "istio galley image"
)
