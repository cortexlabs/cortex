/*
Copyright 2019 Cortex Labs, Inc.

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
	SpotKey                                = "spot"
	InstanceDistributionKey                = "instance_distribution"
	OnDemandBaseCapacityKey                = "on_demand_base_capacity"
	OnDemandPercentageAboveBaseCapacityKey = "on_demand_percentage_above_base_capacity"
	MaxPriceKey                            = "max_price"
	SpotInstancePoolsKey                   = "spot_instance_pools"

	// User facing string
	APIVersionUserFacingKey                          = "cluster version"
	ClusterNameUserFacingKey                         = "cluster name"
	RegionUserFacingKey                              = "region"
	BucketUserFacingKey                              = "bucket"
	SpotUserFacingKey                                = "use spot instances"
	InstanceTypeUserFacingKey                        = "instance type"
	MinInstancesUserFacingKey                        = "min instances"
	MaxInstancesUserFacingKey                        = "max instances"
	InstanceDistributionUserFacingKey                = "instance distribution"
	OnDemandBaseCapacityUserFacingKey                = "on demand base capacity"
	OnDemandPercentageAboveBaseCapacityUserFacingKey = "on demand percentage above base capacity"
	MaxPriceUserFacingKey                            = "max price"
	SpotInstancePoolsUserFacingKey                   = "spot instance pools"
	LogGroupUserFacingKey                            = "log group"
	TelemetryUserFacingKey                           = "telemetry"
	ImagePredictorServeUserFacingKey                 = "predictor serving image"
	ImagePredictorServeGPUUserFacingKey              = "predictor serving gpu image"
	ImageTFServeUserFacingKey                        = "TensorFlow serving image"
	ImageTFServeGPUUserFacingKey                     = "TensorFlow serving gpu image"
	ImageTFAPIUserFacingKey                          = "TensorFlow api image"
	ImageONNXServeUserFacingKey                      = "ONNX serving image"
	ImageONNXServeGPUUserFacingKey                   = "ONNX serving gpu image"
	ImageOperatorUserFacingKey                       = "operator image"
	ImageManagerUserFacingKey                        = "manager image"
	ImageDownloaderUserFacingKey                     = "downloader image"
	ImageClusterAutoscalerUserFacingKey              = "cluster autoscaler image"
	ImageMetricsServerUserFacingKey                  = "metrics server image"
	ImageNvidiaUserFacingKey                         = "NVIDIA image"
	ImageFluentdUserFacingKey                        = "Fluentd image"
	ImageStatsdUserFacingKey                         = "StatsD image"
	ImageIstioProxyUserFacingKey                     = "Istio proxy image"
	ImageIstioPilotUserFacingKey                     = "Istio pilot image"
	ImageIstioCitadelUserFacingKey                   = "Istio citadel image"
	ImageIstioGalleyUserFacingKey                    = "Istio galley image"
)
