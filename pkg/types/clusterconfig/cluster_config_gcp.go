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

import (
	"fmt"
	"strings"

	"github.com/cortexlabs/cortex/pkg/consts"
	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/gcp"
	"github.com/cortexlabs/cortex/pkg/lib/hash"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/prompt"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/lib/slices"
	"github.com/cortexlabs/cortex/pkg/types"
)

var (
	_maxNodePoolLengthWithPrefix = 19                                           // node pool length name limit on GKE
	_maxNodePoolLength           = _maxNodePoolLengthWithPrefix - len("cx-wd-") // or cx-ws-
)

type GCPCoreConfig struct {
	Provider       types.ProviderType `json:"provider" yaml:"provider"`
	Project        string             `json:"project" yaml:"project"`
	Zone           string             `json:"zone" yaml:"zone"`
	ClusterName    string             `json:"cluster_name" yaml:"cluster_name"`
	Telemetry      bool               `json:"telemetry" yaml:"telemetry"`
	Namespace      string             `json:"namespace" yaml:"namespace"`
	IstioNamespace string             `json:"istio_namespace" yaml:"istio_namespace"`
	IsManaged      bool               `json:"is_managed" yaml:"is_managed"`
	Bucket         string             `json:"bucket" yaml:"bucket"`

	ImageOperator                   string `json:"image_operator" yaml:"image_operator"`
	ImageManager                    string `json:"image_manager" yaml:"image_manager"`
	ImageDownloader                 string `json:"image_downloader" yaml:"image_downloader"`
	ImageRequestMonitor             string `json:"image_request_monitor" yaml:"image_request_monitor"`
	ImageClusterAutoscaler          string `json:"image_cluster_autoscaler" yaml:"image_cluster_autoscaler"`
	ImageFluentBit                  string `json:"image_fluent_bit" yaml:"image_fluent_bit"`
	ImageIstioProxy                 string `json:"image_istio_proxy" yaml:"image_istio_proxy"`
	ImageIstioPilot                 string `json:"image_istio_pilot" yaml:"image_istio_pilot"`
	ImageGooglePause                string `json:"image_google_pause" yaml:"image_google_pause"`
	ImagePrometheus                 string `json:"image_prometheus" yaml:"image_prometheus"`
	ImagePrometheusConfigReloader   string `json:"image_prometheus_config_reloader" yaml:"image_prometheus_config_reloader"`
	ImagePrometheusOperator         string `json:"image_prometheus_operator" yaml:"image_prometheus_operator"`
	ImagePrometheusStatsDExporter   string `json:"image_prometheus_statsd_exporter" yaml:"image_prometheus_statsd_exporter"`
	ImagePrometheusDCGMExporter     string `json:"image_prometheus_dcgm_exporter" yaml:"image_prometheus_dcgm_exporter"`
	ImagePrometheusKubeStateMetrics string `json:"image_prometheus_kube_state_metrics" yaml:"image_prometheus_kube_state_metrics"`
	ImagePrometheusNodeExporter     string `json:"image_prometheus_node_exporter" yaml:"image_prometheus_node_exporter"`
	ImageKubeRBACProxy              string `json:"image_kube_rbac_proxy" yaml:"image_kube_rbac_proxy"`
	ImageGrafana                    string `json:"image_grafana" yaml:"image_grafana"`
	ImageEventExporter              string `json:"image_event_exporter" yaml:"image_event_exporter"`
}

type GCPManagedConfig struct {
	NodePools                  []*NodePool        `json:"node_pools" yaml:"node_pools"`
	Network                    *string            `json:"network,omitempty" yaml:"network,omitempty"`
	Subnet                     *string            `json:"subnet,omitempty" yaml:"subnet,omitempty"`
	APILoadBalancerScheme      LoadBalancerScheme `json:"api_load_balancer_scheme" yaml:"api_load_balancer_scheme"`
	OperatorLoadBalancerScheme LoadBalancerScheme `json:"operator_load_balancer_scheme" yaml:"operator_load_balancer_scheme"`
}

type NodePool struct {
	Name                    string  `json:"name" yaml:"name"`
	InstanceType            string  `json:"instance_type" yaml:"instance_type"`
	AcceleratorType         *string `json:"accelerator_type,omitempty" yaml:"accelerator_type,omitempty"`
	AcceleratorsPerInstance *int64  `json:"accelerators_per_instance,omitempty" yaml:"accelerators_per_instance,omitempty"`
	MinInstances            int64   `json:"min_instances" yaml:"min_instances"`
	MaxInstances            int64   `json:"max_instances" yaml:"max_instances"`
	Preemptible             bool    `json:"preemptible" yaml:"preemptible"`
}

type GCPConfig struct {
	GCPCoreConfig    `yaml:",inline"`
	GCPManagedConfig `yaml:",inline"`
}

type InternalGCPConfig struct {
	GCPConfig

	// Populated by operator
	OperatorMetadata
}

// The bare minimum to identify a cluster
type GCPAccessConfig struct {
	ClusterName  string `json:"cluster_name" yaml:"cluster_name"`
	Project      string `json:"project" yaml:"project"`
	Zone         string `json:"zone" yaml:"zone"`
	ImageManager string `json:"image_manager" yaml:"image_manager"`
}

var GCPCoreConfigStructFieldValidations = []*cr.StructFieldValidation{
	{
		StructField: "Provider",
		StringValidation: &cr.StringValidation{
			Validator: specificProviderTypeValidator(types.GCPProviderType),
			Default:   types.GCPProviderType.String(),
		},
		Parser: func(str string) (interface{}, error) {
			return types.ProviderTypeFromString(str), nil
		},
	},
	{
		StructField: "ClusterName",
		StringValidation: &cr.StringValidation{
			Default:   "cortex",
			MaxLength: 40, // this is GKE's limit
			MinLength: 3,
			Validator: validateClusterName,
		},
	},
	{
		StructField: "Project",
		StringValidation: &cr.StringValidation{
			Required: true,
		},
	},
	{
		StructField: "Zone",
		StringValidation: &cr.StringValidation{
			Required: true,
		},
	},
	{
		StructField: "IsManaged",
		BoolValidation: &cr.BoolValidation{
			Default: true,
		},
	},
	{
		StructField: "Namespace",
		StringValidation: &cr.StringValidation{
			Default: "default",
		},
	},
	{
		StructField: "Bucket",
		StringValidation: &cr.StringValidation{
			AllowEmpty:       true,
			TreatNullAsEmpty: true,
		},
	},
	{
		StructField: "IstioNamespace",
		StringValidation: &cr.StringValidation{
			Default: "istio-system",
		},
	},
	{
		StructField: "ImageOperator",
		StringValidation: &cr.StringValidation{
			Default:   "quay.io/cortexlabs/operator:" + consts.CortexVersion,
			Validator: validateImageVersion,
		},
	},
	{
		StructField: "ImageManager",
		StringValidation: &cr.StringValidation{
			Default:   "quay.io/cortexlabs/manager:" + consts.CortexVersion,
			Validator: validateImageVersion,
		},
	},
	{
		StructField: "ImageDownloader",
		StringValidation: &cr.StringValidation{
			Default:   "quay.io/cortexlabs/downloader:" + consts.CortexVersion,
			Validator: validateImageVersion,
		},
	},
	{
		StructField: "ImageRequestMonitor",
		StringValidation: &cr.StringValidation{
			Default:   "quay.io/cortexlabs/request-monitor:" + consts.CortexVersion,
			Validator: validateImageVersion,
		},
	},
	{
		StructField: "ImageClusterAutoscaler",
		StringValidation: &cr.StringValidation{
			Default:   "quay.io/cortexlabs/cluster-autoscaler:" + consts.CortexVersion,
			Validator: validateImageVersion,
		},
	},
	{
		StructField: "ImageFluentBit",
		StringValidation: &cr.StringValidation{
			Default:   "quay.io/cortexlabs/fluent-bit:" + consts.CortexVersion,
			Validator: validateImageVersion,
		},
	},
	{
		StructField: "ImageIstioProxy",
		StringValidation: &cr.StringValidation{
			Default:   "quay.io/cortexlabs/istio-proxy:" + consts.CortexVersion,
			Validator: validateImageVersion,
		},
	},
	{
		StructField: "ImageIstioPilot",
		StringValidation: &cr.StringValidation{
			Default:   "quay.io/cortexlabs/istio-pilot:" + consts.CortexVersion,
			Validator: validateImageVersion,
		},
	},
	{
		StructField: "ImageGooglePause",
		StringValidation: &cr.StringValidation{
			Default:   "quay.io/cortexlabs/google-pause:" + consts.CortexVersion,
			Validator: validateImageVersion,
		},
	},
	{
		StructField: "ImagePrometheus",
		StringValidation: &cr.StringValidation{
			Default:   "quay.io/cortexlabs/prometheus:" + consts.CortexVersion,
			Validator: validateImageVersion,
		},
	},
	{
		StructField: "ImagePrometheusConfigReloader",
		StringValidation: &cr.StringValidation{
			Default:   "quay.io/cortexlabs/prometheus-config-reloader:" + consts.CortexVersion,
			Validator: validateImageVersion,
		},
	},
	{
		StructField: "ImagePrometheusOperator",
		StringValidation: &cr.StringValidation{
			Default:   "quay.io/cortexlabs/prometheus-operator:" + consts.CortexVersion,
			Validator: validateImageVersion,
		},
	},
	{
		StructField: "ImagePrometheusNodeExporter",
		StringValidation: &cr.StringValidation{
			Default:   "quay.io/cortexlabs/prometheus-node-exporter:" + consts.CortexVersion,
			Validator: validateImageVersion,
		},
	},
	{
		StructField: "ImageKubeRBACProxy",
		StringValidation: &cr.StringValidation{
			Default:   "quay.io/cortexlabs/kube-rbac-proxy:" + consts.CortexVersion,
			Validator: validateImageVersion,
		},
	},
	{
		StructField: "ImagePrometheusStatsDExporter",
		StringValidation: &cr.StringValidation{
			Default:   "quay.io/cortexlabs/prometheus-statsd-exporter:" + consts.CortexVersion,
			Validator: validateImageVersion,
		},
	},
	{
		StructField: "ImagePrometheusDCGMExporter",
		StringValidation: &cr.StringValidation{
			Default:   "quay.io/cortexlabs/prometheus-dcgm-exporter:" + consts.CortexVersion,
			Validator: validateImageVersion,
		},
	},
	{
		StructField: "ImagePrometheusKubeStateMetrics",
		StringValidation: &cr.StringValidation{
			Default:   "quay.io/cortexlabs/prometheus-kube-state-metrics:" + consts.CortexVersion,
			Validator: validateImageVersion,
		},
	},
	{
		StructField: "ImageGrafana",
		StringValidation: &cr.StringValidation{
			Default:   "quay.io/cortexlabs/grafana:" + consts.CortexVersion,
			Validator: validateImageVersion,
		},
	},
	{
		StructField: "ImageEventExporter",
		StringValidation: &cr.StringValidation{
			Default:   "quay.io/cortexlabs/event-exporter:" + consts.CortexVersion,
			Validator: validateImageVersion,
		},
	},
	{
		StructField: "Telemetry",
		BoolValidation: &cr.BoolValidation{
			Default: true,
		},
	},
}

var GCPManagedConfigStructFieldValidations = []*cr.StructFieldValidation{
	{
		StructField: "NodePools",
		StructListValidation: &cr.StructListValidation{
			Required: true,
			StructValidation: &cr.StructValidation{
				StructFieldValidations: []*cr.StructFieldValidation{
					{
						StructField: "Name",
						StringValidation: &cr.StringValidation{
							Required:                   true,
							AlphaNumericDashUnderscore: true,
							MaxLength:                  _maxNodePoolLength,
						},
					},
					{
						StructField: "InstanceType",
						StringValidation: &cr.StringValidation{
							Required: true,
						},
					},
					{
						StructField: "AcceleratorType",
						StringPtrValidation: &cr.StringPtrValidation{
							AllowExplicitNull: true,
						},
					},
					{
						StructField: "AcceleratorsPerInstance",
						Int64PtrValidation: &cr.Int64PtrValidation{
							AllowExplicitNull: true,
						},
						DefaultDependentFields: []string{"AcceleratorType"},
						DefaultDependentFieldsFunc: func(vals []interface{}) interface{} {
							acceleratorType := vals[0].(*string)
							if acceleratorType == nil {
								return nil
							}
							return pointer.Int64(1)
						},
					},
					{
						StructField: "MinInstances",
						Int64Validation: &cr.Int64Validation{
							Default:              int64(1),
							GreaterThanOrEqualTo: pointer.Int64(0),
						},
					},
					{
						StructField: "MaxInstances",
						Int64Validation: &cr.Int64Validation{
							Default:     int64(5),
							GreaterThan: pointer.Int64(0),
						},
					},
					{
						StructField: "Preemptible",
						BoolValidation: &cr.BoolValidation{
							Default: false,
						},
					},
				},
			},
		},
	},
	{
		StructField: "Network",
		StringPtrValidation: &cr.StringPtrValidation{
			AllowExplicitNull: true,
		},
	},
	{
		StructField: "Subnet",
		StringPtrValidation: &cr.StringPtrValidation{
			AllowExplicitNull: true,
		},
	},
	{
		StructField: "APILoadBalancerScheme",
		StringValidation: &cr.StringValidation{
			AllowedValues: LoadBalancerSchemeStrings(),
			Default:       InternetFacingLoadBalancerScheme.String(),
		},
		Parser: func(str string) (interface{}, error) {
			return LoadBalancerSchemeFromString(str), nil
		},
	},
	{
		StructField: "OperatorLoadBalancerScheme",
		StringValidation: &cr.StringValidation{
			AllowedValues: LoadBalancerSchemeStrings(),
			Default:       InternetFacingLoadBalancerScheme.String(),
		},
		Parser: func(str string) (interface{}, error) {
			return LoadBalancerSchemeFromString(str), nil
		},
	},
}

var GCPAccessValidation = &cr.StructValidation{
	AllowExtraFields: true,
	StructFieldValidations: []*cr.StructFieldValidation{
		{
			StructField: "ClusterName",
			StringValidation: &cr.StringValidation{
				Default:   "cortex",
				MaxLength: 40, // this is GKE's limit
				MinLength: 3,
				Validator: validateClusterName,
			},
		},
		{
			StructField: "Zone",
			StringValidation: &cr.StringValidation{
				Required: true,
			},
		},
		{
			StructField: "Project",
			StringValidation: &cr.StringValidation{
				Required: true,
			},
		},
		{
			StructField: "ImageManager",
			StringValidation: &cr.StringValidation{
				Default:   "quay.io/cortexlabs/manager:" + consts.CortexVersion,
				Validator: validateImageVersion,
			},
		},
	},
}

func (cc *GCPConfig) ToAccessConfig() GCPAccessConfig {
	return GCPAccessConfig{
		ClusterName:  cc.ClusterName,
		Zone:         cc.Zone,
		Project:      cc.Project,
		ImageManager: cc.ImageManager,
	}
}

func GCPCoreConfigValidations(allowExtraFields bool) *cr.StructValidation {
	return &cr.StructValidation{
		Required:               true,
		StructFieldValidations: GCPCoreConfigStructFieldValidations,
		AllowExtraFields:       allowExtraFields,
	}
}

func GCPManagedConfigValidations(allowExtraFields bool) *cr.StructValidation {
	return &cr.StructValidation{
		Required:               true,
		StructFieldValidations: GCPManagedConfigStructFieldValidations,
		AllowExtraFields:       allowExtraFields,
	}
}

var GCPFullManagedValidation = &cr.StructValidation{
	Required:               true,
	StructFieldValidations: append([]*cr.StructFieldValidation{}, append(GCPCoreConfigStructFieldValidations, GCPManagedConfigStructFieldValidations...)...),
}

var GCPAccessPromptValidation = &cr.PromptValidation{
	SkipNonNilFields: true,
	PromptItemValidations: []*cr.PromptItemValidation{
		{
			StructField: "Project",
			PromptOpts: &prompt.Options{
				Prompt: ProjectUserKey,
			},
			StringPtrValidation: &cr.StringPtrValidation{
				Required: true,
			},
		},
		{
			StructField: "Zone",
			PromptOpts: &prompt.Options{
				Prompt: ZoneUserKey,
			},
			StringPtrValidation: &cr.StringPtrValidation{
				Default: pointer.String("us-east1-c"),
			},
		},
		{
			StructField: "ClusterName",
			PromptOpts: &prompt.Options{
				Prompt: ClusterNameUserKey,
			},
			StringPtrValidation: &cr.StringPtrValidation{
				Default:   pointer.String("cortex"),
				MaxLength: 40, // this is GKE's limit
				MinLength: 3,
				Validator: validateClusterName,
			},
		},
	},
}

// this validates the user-provided cluster config
func (cc *GCPConfig) Validate(GCP *gcp.Client) error {
	fmt.Print("verifying your configuration ...\n\n")

	if validID, err := GCP.IsProjectIDValid(); err != nil {
		return err
	} else if !validID {
		return ErrorGCPInvalidProjectID(cc.Project)
	}

	if validZone, err := GCP.IsZoneValid(cc.Zone); err != nil {
		return err
	} else if !validZone {
		availableZones, err := GCP.GetAvailableZones()
		if err != nil {
			return err
		}
		return ErrorGCPInvalidZone(cc.Zone, availableZones...)
	}

	if cc.Bucket == "" {
		cc.Bucket = GCPBucketName(cc.ClusterName, cc.Project, cc.Zone)
	}

	numNodePools := len(cc.NodePools)
	if numNodePools == 0 {
		return ErrorGCPNoNodePoolSpecified()
	}
	if numNodePools > MaxNodePoolsOrGroups {
		return ErrorGCPMaxNumOfNodePoolsReached(MaxNodePoolsOrGroups)
	}

	npNames := []string{}
	for _, nodePool := range cc.NodePools {
		if !slices.HasString(npNames, nodePool.Name) {
			npNames = append(npNames, nodePool.Name)
		} else {
			return errors.Wrap(ErrorGCPDuplicateNodePoolName(nodePool.Name), NodePoolsKey)
		}

		err := nodePool.validateNodePool(GCP, cc.Zone)
		if err != nil {
			return errors.Wrap(err, NodeGroupsKey, nodePool.Name)
		}

	}

	return nil
}

func (np *NodePool) validateNodePool(GCP *gcp.Client, zone string) error {
	if validInstanceType, err := GCP.IsInstanceTypeAvailable(np.InstanceType, zone); err != nil {
		return err
	} else if !validInstanceType {
		instanceTypes, err := GCP.GetAvailableInstanceTypes(zone)
		if err != nil {
			return err
		}
		return ErrorGCPInvalidInstanceType(np.InstanceType, instanceTypes...)
	}

	if np.AcceleratorType == nil && np.AcceleratorsPerInstance != nil {
		return ErrorDependentFieldMustBeSpecified(AcceleratorsPerInstanceKey, AcceleratorTypeKey)
	}

	if np.AcceleratorType != nil {
		if np.AcceleratorsPerInstance == nil {
			return ErrorDependentFieldMustBeSpecified(AcceleratorTypeKey, AcceleratorsPerInstanceKey)
		}
		if validAccelerator, err := GCP.IsAcceleratorTypeAvailable(*np.AcceleratorType, zone); err != nil {
			return err
		} else if !validAccelerator {
			availableAcceleratorsInZone, err := GCP.GetAvailableAcceleratorTypes(zone)
			if err != nil {
				return err
			}
			allAcceleratorTypes, err := GCP.GetAvailableAcceleratorTypesForAllZones()
			if err != nil {
				return err
			}

			var availableZonesForAccelerator []string
			if slices.HasString(allAcceleratorTypes, *np.AcceleratorType) {
				availableZonesForAccelerator, err = GCP.GetAvailableZonesForAccelerator(*np.AcceleratorType)
				if err != nil {
					return err
				}
			}
			return ErrorGCPInvalidAcceleratorType(*np.AcceleratorType, zone, availableAcceleratorsInZone, availableZonesForAccelerator)
		}

		// according to https://cloud.google.com/kubernetes-engine/docs/how-to/gpus
		var compatibleInstances []string
		var err error
		if strings.HasSuffix(*np.AcceleratorType, "a100") {
			compatibleInstances, err = GCP.GetInstanceTypesWithPrefix("a2", zone)
		} else {
			compatibleInstances, err = GCP.GetInstanceTypesWithPrefix("n1", zone)
		}
		if err != nil {
			return err
		}
		if !slices.HasString(compatibleInstances, np.InstanceType) {
			return ErrorGCPIncompatibleInstanceTypeWithAccelerator(np.InstanceType, *np.AcceleratorType, zone, compatibleInstances)
		}
	}

	return nil
}

func (cc *GCPCoreConfig) TelemetryEvent() map[string]interface{} {
	event := map[string]interface{}{
		"provider":   types.GCPProviderType,
		"is_managed": cc.IsManaged,
	}

	if cc.ClusterName != "cortex" {
		event["cluster_name._is_custom"] = true
	}

	event["zone"] = cc.Zone

	if cc.Namespace != "default" {
		event["namespace._is_custom"] = true
	}
	if cc.IstioNamespace != "istio-system" {
		event["istio_namespace._is_custom"] = true
	}

	if !strings.HasPrefix(cc.ImageOperator, "cortexlabs/") {
		event["image_operator._is_custom"] = true
	}
	if !strings.HasPrefix(cc.ImageManager, "cortexlabs/") {
		event["image_manager._is_custom"] = true
	}
	if !strings.HasPrefix(cc.ImageDownloader, "cortexlabs/") {
		event["image_downloader._is_custom"] = true
	}
	if !strings.HasPrefix(cc.ImageRequestMonitor, "cortexlabs/") {
		event["image_request_monitor._is_custom"] = true
	}
	if !strings.HasPrefix(cc.ImageClusterAutoscaler, "cortexlabs/") {
		event["image_cluster_autoscaler._is_custom"] = true
	}
	if !strings.HasPrefix(cc.ImageFluentBit, "cortexlabs/") {
		event["image_fluent_bit._is_custom"] = true
	}
	if !strings.HasPrefix(cc.ImageIstioProxy, "cortexlabs/") {
		event["image_istio_proxy._is_custom"] = true
	}
	if !strings.HasPrefix(cc.ImageIstioPilot, "cortexlabs/") {
		event["image_istio_pilot._is_custom"] = true
	}
	if !strings.HasPrefix(cc.ImageGooglePause, "cortexlabs/") {
		event["image_google_pause._is_custom"] = true
	}
	if strings.HasPrefix(cc.ImagePrometheus, "cortexlabs/") {
		event["image_prometheus._is_custom"] = true
	}
	if strings.HasPrefix(cc.ImagePrometheusConfigReloader, "cortexlabs/") {
		event["image_prometheus_config_reloader._is_custom"] = true
	}
	if strings.HasPrefix(cc.ImagePrometheusOperator, "cortexlabs/") {
		event["image_prometheus_operator._is_custom"] = true
	}
	if strings.HasPrefix(cc.ImagePrometheusStatsDExporter, "cortexlabs/") {
		event["image_prometheus_statsd_exporter._is_custom"] = true
	}
	if strings.HasPrefix(cc.ImagePrometheusDCGMExporter, "cortexlabs/") {
		event["image_prometheus_dcgm_exporter._is_custom"] = true
	}
	if strings.HasPrefix(cc.ImagePrometheusKubeStateMetrics, "cortexlabs/") {
		event["image_prometheus_kube_state_metrics._is_custom"] = true
	}
	if strings.HasPrefix(cc.ImagePrometheusNodeExporter, "cortexlabs/") {
		event["image_prometheus_node_exporter._is_custom"] = true
	}
	if strings.HasPrefix(cc.ImageKubeRBACProxy, "cortexlabs/") {
		event["image_kube_rbac_proxy._is_custom"] = true
	}
	if strings.HasPrefix(cc.ImageGrafana, "cortexlabs/") {
		event["image_grafana._is_custom"] = true
	}
	if strings.HasPrefix(cc.ImageEventExporter, "cortexlabs/") {
		event["image_event_exporter._is_custom"] = true
	}
	return event
}

func (cc *GCPManagedConfig) TelemetryEvent() map[string]interface{} {
	event := map[string]interface{}{}

	onDemandInstanceTypes := strset.New()
	preemptibleInstanceTypes := strset.New()
	var totalMinSize, totalMaxSize int

	event["node_pools._len"] = len(cc.NodePools)
	for _, np := range cc.NodePools {
		nodePoolKey := func(field string) string {
			lifecycle := "on_demand"
			if np.Preemptible {
				lifecycle = "preemptible"
			}
			return fmt.Sprintf("node_pools.%s-%s.%s", np.InstanceType, lifecycle, field)
		}
		event[nodePoolKey("_is_defined")] = true
		event[nodePoolKey("name")] = np.Name
		event[nodePoolKey("instance_type")] = np.InstanceType
		event[nodePoolKey("min_instances")] = np.MinInstances
		event[nodePoolKey("max_instances")] = np.MaxInstances

		if !np.Preemptible {
			onDemandInstanceTypes.Add(np.InstanceType)
		} else {
			preemptibleInstanceTypes.Add(np.InstanceType)
		}
		if np.AcceleratorType != nil {
			event[nodePoolKey("accelerator_type._is_defined")] = true
			event[nodePoolKey("accelerator_type")] = *np.AcceleratorType
		}
		if np.AcceleratorsPerInstance != nil {
			event[nodePoolKey("accelerators_per_instance._is_defined")] = true
			event[nodePoolKey("accelerators_per_instance")] = *np.AcceleratorsPerInstance
		}

		event[nodePoolKey("preemptible")] = np.Preemptible

		totalMinSize += int(np.MinInstances)
		totalMaxSize += int(np.MaxInstances)
	}

	event["node_pools._total_min_size"] = totalMinSize
	event["node_pools._total_max_size"] = totalMaxSize
	event["node_pools._on_demand_instances"] = onDemandInstanceTypes.Slice()
	event["node_pools._spot_instances"] = preemptibleInstanceTypes.Slice()
	event["node_pools._instances"] = strset.Union(onDemandInstanceTypes, preemptibleInstanceTypes).Slice()

	if cc.Network != nil {
		event["network._is_defined"] = true
	}
	if cc.Subnet != nil {
		event["subnet._is_defined"] = true
	}
	event["api_load_balancer_scheme"] = cc.APILoadBalancerScheme
	event["operator_load_balancer_scheme"] = cc.OperatorLoadBalancerScheme

	return event
}

func GCPBucketName(clusterName string, project string, zone string) string {
	bucketID := hash.String(project + zone)[:8] // this is to "guarantee" a globally unique name
	return clusterName + "-" + bucketID
}
