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

import (
	"github.com/cortexlabs/cortex/pkg/types"
)

type GCPConfig struct {
	Provider            types.ProviderType `json:"provider" yaml:"provider"`
	Bucket              string             `json:"bucket" yaml:"bucket"`
	Project             string             `json:"project" yaml:"project"`
	ClusterName         string             `json:"cluster_name" yaml:"cluster_name"`
	Zone                string             `json:"zone" yaml:"zone"`
	InstanceType        string             `json:"instance_type" yaml:"instance_type"`
	AcceleratorType     *string            `json:"accelerator_type" yaml:"accelerator_type"`
	MinInstances        int                `json:"min_instances" yaml:"min_instances"`
	MaxInstances        int                `json:"max_instances" yaml:"max_instances"`
	Telemetry           bool               `json:"telemetry" yaml:"telemetry"`
	ImageOperator       string             `json:"image_operator" yaml:"image_operator"`
	ImageManager        string             `json:"image_manager" yaml:"image_manager"`
	ImageDownloader     string             `json:"image_downloader" yaml:"image_downloader"`
	ImageRequestMonitor string             `json:"image_request_monitor" yaml:"image_request_monitor"`
	ImageMetricsServer  string             `json:"image_metrics_server" yaml:"image_metrics_server"`
	ImageFluentd        string             `json:"image_fluentd" yaml:"image_fluentd"`
	ImageIstioProxy     string             `json:"image_istio_proxy" yaml:"image_istio_proxy"`
	ImageIstioPilot     string             `json:"image_istio_pilot" yaml:"image_istio_pilot"`
}

type InternalGCPConfig struct {
	GCPConfig

	// Populated by operator
	ID                string `json:"id"`
	APIVersion        string `json:"api_version"`
	OperatorInCluster bool   `json:"operator_in_cluster"`
}
