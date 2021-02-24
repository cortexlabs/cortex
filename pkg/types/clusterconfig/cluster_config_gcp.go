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
	"github.com/cortexlabs/cortex/pkg/lib/slices"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/table"
	"github.com/cortexlabs/cortex/pkg/types"
)

type GCPCoreConfig struct {
	Provider       types.ProviderType `json:"provider" yaml:"provider"`
	Project        *string            `json:"project" yaml:"project"`
	Zone           *string            `json:"zone" yaml:"zone"`
	ClusterName    string             `json:"cluster_name" yaml:"cluster_name"`
	Telemetry      bool               `json:"telemetry" yaml:"telemetry"`
	Namespace      string             `json:"namespace" yaml:"namespace"`
	IstioNamespace string             `json:"istio_namespace" yaml:"istio_namespace"`
	IsManaged      bool               `json:"is_managed" yaml:"is_managed"`
	Bucket         string             `json:"bucket" yaml:"bucket"`

	ImageOperator                 string `json:"image_operator" yaml:"image_operator"`
	ImageManager                  string `json:"image_manager" yaml:"image_manager"`
	ImageDownloader               string `json:"image_downloader" yaml:"image_downloader"`
	ImageRequestMonitor           string `json:"image_request_monitor" yaml:"image_request_monitor"`
	ImageClusterAutoscaler        string `json:"image_cluster_autoscaler" yaml:"image_cluster_autoscaler"`
	ImageFluentBit                string `json:"image_fluent_bit" yaml:"image_fluent_bit"`
	ImageIstioProxy               string `json:"image_istio_proxy" yaml:"image_istio_proxy"`
	ImageIstioPilot               string `json:"image_istio_pilot" yaml:"image_istio_pilot"`
	ImageGooglePause              string `json:"image_google_pause" yaml:"image_google_pause"`
	ImagePrometheus               string `json:"image_prometheus" yaml:"image_prometheus"`
	ImagePrometheusConfigReloader string `json:"image_prometheus_config_reloader" yaml:"image_prometheus_config_reloader"`
	ImagePrometheusOperator       string `json:"image_prometheus_operator" yaml:"image_prometheus_operator"`
	ImagePrometheusStatsDExporter string `json:"image_prometheus_statsd_exporter" yaml:"image_prometheus_statsd_exporter"`
	ImageGrafana                  string `json:"image_grafana" yaml:"image_grafana"`
	ImageEventExporter            string `json:"image_event_exporter" yaml:"image_event_exporter"`
}

type GCPManagedConfig struct {
	InstanceType               *string            `json:"instance_type" yaml:"instance_type"`
	AcceleratorType            *string            `json:"accelerator_type" yaml:"accelerator_type"`
	AcceleratorsPerInstance    *int64             `json:"accelerators_per_instance" yaml:"accelerators_per_instance"`
	Network                    *string            `json:"network" yaml:"network"`
	Subnet                     *string            `json:"subnet" yaml:"subnet"`
	APILoadBalancerScheme      LoadBalancerScheme `json:"api_load_balancer_scheme" yaml:"api_load_balancer_scheme"`
	OperatorLoadBalancerScheme LoadBalancerScheme `json:"operator_load_balancer_scheme" yaml:"operator_load_balancer_scheme"`
	MinInstances               *int64             `json:"min_instances" yaml:"min_instances"`
	MaxInstances               *int64             `json:"max_instances" yaml:"max_instances"`
	Preemptible                bool               `json:"preemptible" yaml:"preemptible"`
	OnDemandBackup             bool               `json:"on_demand_backup" yaml:"on_demand_backup"`
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
	ClusterName  *string `json:"cluster_name" yaml:"cluster_name"`
	Project      *string `json:"project" yaml:"project"`
	Zone         *string `json:"zone" yaml:"zone"`
	ImageManager string  `json:"image_manager" yaml:"image_manager"`
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
			MaxLength: 63,
			MinLength: 3,
			Validator: validateClusterName,
		},
	},
	{
		StructField:         "Project",
		StringPtrValidation: &cr.StringPtrValidation{},
	},
	{
		StructField:         "Zone",
		StringPtrValidation: &cr.StringPtrValidation{},
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
		StructField: "ImagePrometheusStatsDExporter",
		StringValidation: &cr.StringValidation{
			Default:   "quay.io/cortexlabs/prometheus-statsd-exporter:" + consts.CortexVersion,
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
		StructField:         "InstanceType",
		StringPtrValidation: &cr.StringPtrValidation{},
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
	{
		StructField: "MinInstances",
		Int64PtrValidation: &cr.Int64PtrValidation{
			GreaterThanOrEqualTo: pointer.Int64(0),
		},
	},
	{
		StructField: "MaxInstances",
		Int64PtrValidation: &cr.Int64PtrValidation{
			GreaterThan: pointer.Int64(0),
		},
	},
	{
		StructField: "Preemptible",
		BoolValidation: &cr.BoolValidation{
			Default: false,
		},
	},
	{
		StructField:            "OnDemandBackup",
		DefaultDependentFields: []string{"Preemptible"},
		DefaultDependentFieldsFunc: func(vals []interface{}) interface{} {
			return vals[0].(bool)
		},
		BoolValidation: &cr.BoolValidation{},
	},
}

var GCPAccessValidation = &cr.StructValidation{
	AllowExtraFields: true,
	StructFieldValidations: []*cr.StructFieldValidation{
		{
			StructField: "ClusterName",
			StringPtrValidation: &cr.StringPtrValidation{
				MaxLength: 63,
				MinLength: 3,
				Validator: validateClusterName,
			},
		},
		{
			StructField:         "Zone",
			StringPtrValidation: &cr.StringPtrValidation{},
		},
		{
			StructField:         "Project",
			StringPtrValidation: &cr.StringPtrValidation{},
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
	clusterName := cc.ClusterName
	zone := *cc.Zone
	project := *cc.Project
	return GCPAccessConfig{
		ClusterName:  &clusterName,
		Zone:         &zone,
		Project:      &project,
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
				MaxLength: 63,
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
		return ErrorGCPInvalidProjectID(*cc.Project)
	}

	if validZone, err := GCP.IsZoneValid(*cc.Zone); err != nil {
		return err
	} else if !validZone {
		availableZones, err := GCP.GetAvailableZones()
		if err != nil {
			return err
		}
		return ErrorGCPInvalidZone(*cc.Zone, availableZones...)
	}

	if cc.Bucket == "" {
		cc.Bucket = GCPBucketName(cc.ClusterName, *cc.Project, *cc.Zone)
	}

	if validInstanceType, err := GCP.IsInstanceTypeAvailable(*cc.InstanceType, *cc.Zone); err != nil {
		return err
	} else if !validInstanceType {
		instanceTypes, err := GCP.GetAvailableInstanceTypes(*cc.Zone)
		if err != nil {
			return err
		}
		return ErrorGCPInvalidInstanceType(*cc.InstanceType, instanceTypes...)
	}

	if cc.AcceleratorType == nil && cc.AcceleratorsPerInstance != nil {
		return ErrorDependentFieldMustBeSpecified(AcceleratorsPerInstanceKey, AcceleratorTypeKey)
	}

	if cc.AcceleratorType != nil {
		if cc.AcceleratorsPerInstance == nil {
			return ErrorDependentFieldMustBeSpecified(AcceleratorTypeKey, AcceleratorsPerInstanceKey)
		}
		if validAccelerator, err := GCP.IsAcceleratorTypeAvailable(*cc.AcceleratorType, *cc.Zone); err != nil {
			return err
		} else if !validAccelerator {
			availableAcceleratorsInZone, err := GCP.GetAvailableAcceleratorTypes(*cc.Zone)
			if err != nil {
				return err
			}
			allAcceleratorTypes, err := GCP.GetAvailableAcceleratorTypesForAllZones()
			if err != nil {
				return err
			}

			var availableZonesForAccelerator []string
			if slices.HasString(allAcceleratorTypes, *cc.AcceleratorType) {
				availableZonesForAccelerator, err = GCP.GetAvailableZonesForAccelerator(*cc.AcceleratorType)
				if err != nil {
					return err
				}
			}
			return ErrorGCPInvalidAcceleratorType(*cc.AcceleratorType, *cc.Zone, availableAcceleratorsInZone, availableZonesForAccelerator)
		}

		// according to https://cloud.google.com/kubernetes-engine/docs/how-to/gpus
		var compatibleInstances []string
		var err error
		if strings.HasSuffix(*cc.AcceleratorType, "a100") {
			compatibleInstances, err = GCP.GetInstanceTypesWithPrefix("a2", *cc.Zone)
		} else {
			compatibleInstances, err = GCP.GetInstanceTypesWithPrefix("n1", *cc.Zone)
		}
		if err != nil {
			return err
		}
		if !slices.HasString(compatibleInstances, *cc.InstanceType) {
			return ErrorGCPIncompatibleInstanceTypeWithAccelerator(*cc.InstanceType, *cc.AcceleratorType, *cc.Zone, compatibleInstances)
		}
	}

	if !cc.Preemptible && cc.OnDemandBackup {
		return ErrorFieldConfigurationDependentOnCondition(OnDemandBackupKey, s.Bool(cc.OnDemandBackup), PreemptibleKey, s.Bool(cc.Preemptible))
	}

	return nil
}

func applyGCPPromptDefaults(defaults GCPConfig) *GCPConfig {
	defaultConfig := &GCPConfig{
		GCPCoreConfig: GCPCoreConfig{
			Zone: pointer.String("us-east1-c"),
		},
		GCPManagedConfig: GCPManagedConfig{
			InstanceType: pointer.String("n1-standard-2"),
			MinInstances: pointer.Int64(1),
			MaxInstances: pointer.Int64(5),
		},
	}

	if defaults.Zone != nil {
		defaultConfig.Zone = defaults.Zone
	}
	if defaults.InstanceType != nil {
		defaultConfig.InstanceType = defaults.InstanceType
	}
	if defaults.MinInstances != nil {
		defaultConfig.MinInstances = defaults.MinInstances
	}
	if defaults.MaxInstances != nil {
		defaultConfig.MaxInstances = defaults.MaxInstances
	}

	return defaultConfig
}

func InstallGCPPrompt(clusterConfig *GCPConfig, disallowPrompt bool) error {
	defaults := applyGCPPromptDefaults(*clusterConfig)

	if disallowPrompt {
		if clusterConfig.Project == nil {
			return ErrorGCPProjectMustBeSpecified()
		}

		if clusterConfig.Zone == nil {
			clusterConfig.Zone = defaults.Zone
		}
		if clusterConfig.InstanceType == nil {
			clusterConfig.InstanceType = defaults.InstanceType
		}
		if clusterConfig.MinInstances == nil {
			clusterConfig.MinInstances = defaults.MinInstances
		}
		if clusterConfig.MaxInstances == nil {
			clusterConfig.MaxInstances = defaults.MaxInstances
		}
		return nil
	}

	remainingPrompts := &cr.PromptValidation{
		SkipNonEmptyFields: true,
		PromptItemValidations: []*cr.PromptItemValidation{
			{
				StructField: "InstanceType",
				PromptOpts: &prompt.Options{
					Prompt: "instance type",
				},
				StringPtrValidation: &cr.StringPtrValidation{
					Required: true,
					Default:  defaults.InstanceType,
				},
			},
			{
				StructField: "MinInstances",
				PromptOpts: &prompt.Options{
					Prompt: "min instances",
				},
				Int64PtrValidation: &cr.Int64PtrValidation{
					Required:             true,
					Default:              defaults.MinInstances,
					GreaterThanOrEqualTo: pointer.Int64(0),
				},
			},
			{
				StructField: "MaxInstances",
				PromptOpts: &prompt.Options{
					Prompt: "max instances",
				},
				Int64PtrValidation: &cr.Int64PtrValidation{
					Required:    true,
					Default:     defaults.MaxInstances,
					GreaterThan: pointer.Int64(0),
				},
			},
		},
	}

	err := cr.ReadPrompt(clusterConfig, remainingPrompts)
	if err != nil {
		return err
	}

	return nil
}

// This does not set defaults for fields that are prompted from the user
func SetGCPDefaults(cc *GCPConfig) error {
	var emptyMap interface{} = map[interface{}]interface{}{}
	errs := cr.Struct(cc, emptyMap, GCPFullManagedValidation)
	if errors.HasError(errs) {
		return errors.FirstError(errs...)
	}

	return nil
}

func DefaultGCPAccessConfig() (*GCPAccessConfig, error) {
	accessConfig := &GCPAccessConfig{}
	var emptyMap interface{} = map[interface{}]interface{}{}
	errs := cr.Struct(accessConfig, emptyMap, GCPAccessValidation)
	if errors.HasError(errs) {
		return nil, errors.FirstError(errs...)
	}
	return accessConfig, nil
}

func (cc *InternalGCPConfig) UserTable() table.KeyValuePairs {
	var items table.KeyValuePairs

	items.Add(APIVersionUserKey, cc.APIVersion)
	items.AddAll(cc.GCPConfig.UserTable())
	return items
}

func (cc *InternalGCPConfig) UserStr() string {
	return cc.UserTable().String()
}

func (cc *GCPCoreConfig) UserTable() table.KeyValuePairs {
	var items table.KeyValuePairs

	items.Add(ClusterNameUserKey, cc.ClusterName)
	items.Add(ProjectUserKey, *cc.Project)
	items.Add(ZoneUserKey, *cc.Zone)
	items.Add(TelemetryUserKey, cc.Telemetry)
	items.Add(ImageOperatorUserKey, cc.ImageOperator)
	items.Add(ImageManagerUserKey, cc.ImageManager)
	items.Add(ImageDownloaderUserKey, cc.ImageDownloader)
	items.Add(ImageRequestMonitorUserKey, cc.ImageRequestMonitor)
	items.Add(ImageClusterAutoscalerUserKey, cc.ImageClusterAutoscaler)
	items.Add(ImageFluentBitUserKey, cc.ImageFluentBit)
	items.Add(ImageIstioProxyUserKey, cc.ImageIstioProxy)
	items.Add(ImageIstioPilotUserKey, cc.ImageIstioPilot)
	items.Add(ImageGooglePauseUserKey, cc.ImageGooglePause)
	items.Add(ImagePrometheusUserKey, cc.ImagePrometheus)
	items.Add(ImagePrometheusConfigReloaderUserKey, cc.ImagePrometheusConfigReloader)
	items.Add(ImagePrometheusOperatorUserKey, cc.ImagePrometheusOperator)
	items.Add(ImagePrometheusStatsDExporterUserKey, cc.ImagePrometheusStatsDExporter)
	items.Add(ImageGrafanaUserKey, cc.ImageGrafana)
	items.Add(ImageEventExporterUserKey, cc.ImageEventExporter)

	return items
}

func (cc *GCPManagedConfig) UserTable() table.KeyValuePairs {
	var items table.KeyValuePairs

	items.Add(InstanceTypeUserKey, *cc.InstanceType)
	items.Add(MinInstancesUserKey, *cc.MinInstances)
	items.Add(MaxInstancesUserKey, *cc.MaxInstances)
	if cc.AcceleratorType != nil {
		items.Add(AcceleratorTypeUserKey, *cc.AcceleratorType)
	}
	if cc.AcceleratorsPerInstance != nil {
		items.Add(AcceleratorsPerInstanceUserKey, *cc.AcceleratorsPerInstance)
	}
	items.Add(PreemptibleUserKey, s.YesNo(cc.Preemptible))
	items.Add(OnDemandBackupUserKey, s.YesNo(cc.OnDemandBackup))
	if cc.Network != nil {
		items.Add(NetworkUserKey, *cc.Network)
	}
	if cc.Subnet != nil {
		items.Add(SubnetUserKey, *cc.Subnet)
	}
	items.Add(APILoadBalancerSchemeUserKey, cc.APILoadBalancerScheme)
	items.Add(OperatorLoadBalancerSchemeUserKey, cc.OperatorLoadBalancerScheme)

	return items
}

func (cc *GCPConfig) UserTable() table.KeyValuePairs {
	var items *table.KeyValuePairs = &table.KeyValuePairs{}
	items.AddAll(cc.GCPCoreConfig.UserTable())
	if cc.GCPCoreConfig.IsManaged {
		items.AddAll(cc.GCPManagedConfig.UserTable())
	}

	return *items
}

func (cc *GCPConfig) UserStr() string {
	return cc.UserTable().String()
}

func (cc *GCPCoreConfig) TelemetryEvent() map[string]interface{} {
	event := map[string]interface{}{
		"provider":   types.GCPProviderType,
		"is_managed": cc.IsManaged,
	}

	if cc.ClusterName != "cortex" {
		event["cluster_name._is_custom"] = true
	}

	if cc.Zone != nil {
		event["zone._is_defined"] = true
		event["zone"] = *cc.Zone
	}
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

	if cc.InstanceType != nil {
		event["instance_type._is_defined"] = true
		event["instance_type"] = *cc.InstanceType
	}
	if cc.AcceleratorType != nil {
		event["accelerator_type._is_defined"] = true
		event["accelerator_type"] = *cc.AcceleratorType
	}
	if cc.AcceleratorsPerInstance != nil {
		event["accelerators_per_instance._is_defined"] = true
		event["accelerators_per_instance"] = *cc.AcceleratorsPerInstance
	}
	if cc.Network != nil {
		event["network._is_defined"] = true
	}
	if cc.Subnet != nil {
		event["subnet._is_defined"] = true
	}
	event["api_load_balancer_scheme"] = cc.APILoadBalancerScheme
	event["operator_load_balancer_scheme"] = cc.OperatorLoadBalancerScheme
	if cc.MinInstances != nil {
		event["min_instances._is_defined"] = true
		event["min_instances"] = *cc.MinInstances
	}
	if cc.MaxInstances != nil {
		event["max_instances._is_defined"] = true
		event["max_instances"] = *cc.MaxInstances
	}

	event["preemptible"] = cc.Preemptible
	event["on_demand_backup"] = cc.OnDemandBackup

	return event
}

func GCPBucketName(clusterName string, project string, zone string) string {
	bucketID := hash.String(project + zone)[:10]
	return clusterName + "-" + bucketID
}
