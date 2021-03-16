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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"math"
	"net"
	"net/http"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/hash"
	libmath "github.com/cortexlabs/cortex/pkg/lib/math"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/lib/slices"
	"github.com/cortexlabs/cortex/pkg/types"
	"github.com/cortexlabs/yaml"
)

const (
	// the s3 url should be used (rather than the cloudfront URL) to avoid caching
	_cniSupportedInstancesURL = "https://cortex-public.s3-us-west-2.amazonaws.com/cli-assets/cni_supported_instances.txt"
)

var (
	_maxNodeGroupLengthWithPrefix = 19                                            // node pool length name limit on GKE, using the same on AWS for consistency reasons
	_maxNodeGroupLength           = _maxNodeGroupLengthWithPrefix - len("cx-wd-") // or cx-ws-
	_maxInstancePools             = 20
	_cachedCNISupportedInstances  *string
	// This regex is stricter than the actual S3 rules
	_strictS3BucketRegex = regexp.MustCompile(`^([a-z0-9])+(-[a-z0-9]+)*$`)
	_defaultIAMPolicies  = []string{"arn:aws:iam::aws:policy/AmazonS3FullAccess"}
)

type CoreConfig struct {
	Bucket         string             `json:"bucket" yaml:"bucket"`
	ClusterName    string             `json:"cluster_name" yaml:"cluster_name"`
	Region         string             `json:"region" yaml:"region"`
	Provider       types.ProviderType `json:"provider" yaml:"provider"`
	Telemetry      bool               `json:"telemetry" yaml:"telemetry"`
	IsManaged      bool               `json:"is_managed" yaml:"is_managed"`
	Namespace      string             `json:"namespace" yaml:"namespace"`
	IstioNamespace string             `json:"istio_namespace" yaml:"istio_namespace"`

	ImageOperator                   string `json:"image_operator" yaml:"image_operator"`
	ImageManager                    string `json:"image_manager" yaml:"image_manager"`
	ImageDownloader                 string `json:"image_downloader" yaml:"image_downloader"`
	ImageRequestMonitor             string `json:"image_request_monitor" yaml:"image_request_monitor"`
	ImageAsyncGateway               string `json:"image_async_gateway" yaml:"image_async_gateway"`
	ImageClusterAutoscaler          string `json:"image_cluster_autoscaler" yaml:"image_cluster_autoscaler"`
	ImageMetricsServer              string `json:"image_metrics_server" yaml:"image_metrics_server"`
	ImageInferentia                 string `json:"image_inferentia" yaml:"image_inferentia"`
	ImageNeuronRTD                  string `json:"image_neuron_rtd" yaml:"image_neuron_rtd"`
	ImageNvidia                     string `json:"image_nvidia" yaml:"image_nvidia"`
	ImageFluentBit                  string `json:"image_fluent_bit" yaml:"image_fluent_bit"`
	ImageIstioProxy                 string `json:"image_istio_proxy" yaml:"image_istio_proxy"`
	ImageIstioPilot                 string `json:"image_istio_pilot" yaml:"image_istio_pilot"`
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

type ManagedConfig struct {
	NodeGroups                 []*NodeGroup       `json:"node_groups" yaml:"node_groups"`
	Tags                       map[string]string  `json:"tags" yaml:"tags"`
	AvailabilityZones          []string           `json:"availability_zones" yaml:"availability_zones"`
	SSLCertificateARN          *string            `json:"ssl_certificate_arn,omitempty" yaml:"ssl_certificate_arn,omitempty"`
	IAMPolicyARNs              []string           `json:"iam_policy_arns" yaml:"iam_policy_arns"`
	SubnetVisibility           SubnetVisibility   `json:"subnet_visibility" yaml:"subnet_visibility"`
	Subnets                    []*Subnet          `json:"subnets,omitempty" yaml:"subnets,omitempty"`
	NATGateway                 NATGateway         `json:"nat_gateway" yaml:"nat_gateway"`
	APILoadBalancerScheme      LoadBalancerScheme `json:"api_load_balancer_scheme" yaml:"api_load_balancer_scheme"`
	OperatorLoadBalancerScheme LoadBalancerScheme `json:"operator_load_balancer_scheme" yaml:"operator_load_balancer_scheme"`
	VPCCIDR                    *string            `json:"vpc_cidr,omitempty" yaml:"vpc_cidr,omitempty"`
	CortexPolicyARN            string             `json:"cortex_policy_arn" yaml:"cortex_policy_arn"` // this field is not user facing
}

type NodeGroup struct {
	Name               string      `json:"name" yaml:"name"`
	InstanceType       string      `json:"instance_type" yaml:"instance_type"`
	MinInstances       int64       `json:"min_instances" yaml:"min_instances"`
	MaxInstances       int64       `json:"max_instances" yaml:"max_instances"`
	InstanceVolumeSize int64       `json:"instance_volume_size" yaml:"instance_volume_size"`
	InstanceVolumeType VolumeType  `json:"instance_volume_type" yaml:"instance_volume_type"`
	InstanceVolumeIOPS *int64      `json:"instance_volume_iops" yaml:"instance_volume_iops"`
	Spot               bool        `json:"spot" yaml:"spot"`
	SpotConfig         *SpotConfig `json:"spot_config" yaml:"spot_config"`
}

type SpotConfig struct {
	InstanceDistribution                []string `json:"instance_distribution" yaml:"instance_distribution"`
	OnDemandBaseCapacity                *int64   `json:"on_demand_base_capacity" yaml:"on_demand_base_capacity"`
	OnDemandPercentageAboveBaseCapacity *int64   `json:"on_demand_percentage_above_base_capacity" yaml:"on_demand_percentage_above_base_capacity"`
	MaxPrice                            *float64 `json:"max_price" yaml:"max_price"`
	InstancePools                       *int64   `json:"instance_pools" yaml:"instance_pools"`
}

type Subnet struct {
	AvailabilityZone string `json:"availability_zone" yaml:"availability_zone"`
	SubnetID         string `json:"subnet_id" yaml:"subnet_id"`
}

type Config struct {
	CoreConfig    `yaml:",inline"`
	ManagedConfig `yaml:",inline"`
}

type OperatorMetadata struct {
	APIVersion          string `json:"api_version"`
	OperatorID          string `json:"operator_id"`
	ClusterID           string `json:"cluster_id"`
	IsOperatorInCluster bool   `json:"is_operator_in_cluster"`
}

type InternalConfig struct {
	Config

	// Populated by operator
	OperatorMetadata

	InstancesMetadata []aws.InstanceMetadata `json:"instance_metadata"`
}

// The bare minimum to identify a cluster
type AccessConfig struct {
	ClusterName  string `json:"cluster_name" yaml:"cluster_name"`
	Region       string `json:"region" yaml:"region"`
	ImageManager string `json:"image_manager" yaml:"image_manager"`
}

func ValidateRegion(region string) error {
	if !aws.EKSSupportedRegions.Has(region) {
		return ErrorInvalidRegion(region)
	}
	return nil
}

func RegionValidator(region string) (string, error) {
	if err := ValidateRegion(region); err != nil {
		return "", err
	}
	return region, nil
}

func (cc *Config) DeepCopy() (Config, error) {
	bytes, err := yaml.Marshal(cc)
	if err != nil {
		return Config{}, err
	}

	deepCopied := Config{}
	err = yaml.Unmarshal(bytes, &deepCopied)
	if err != nil {
		return Config{}, err
	}

	return deepCopied, nil
}

func (cc *Config) Hash() (string, error) {
	bytes, err := yaml.Marshal(cc)
	if err != nil {
		return "", err
	}

	hash := sha256.New()
	hash.Write(bytes)
	return hex.EncodeToString(hash.Sum(nil)), nil
}

var CoreConfigStructFieldValidations = []*cr.StructFieldValidation{
	{
		StructField: "Provider",
		StringValidation: &cr.StringValidation{
			Validator: specificProviderTypeValidator(types.AWSProviderType),
			Default:   types.AWSProviderType.String(),
		},
		Parser: func(str string) (interface{}, error) {
			return types.ProviderTypeFromString(str), nil
		},
	},
	{
		StructField: "ClusterName",
		StringValidation: &cr.StringValidation{
			Default:   "cortex",
			MaxLength: 54, // leaves room for 8 char uniqueness string (and "-") for bucket name (63 chars max)
			MinLength: 3,
			Validator: validateClusterName,
		},
	},
	{
		StructField: "Region",
		StringValidation: &cr.StringValidation{
			Required:  true,
			MinLength: 1,
			Validator: RegionValidator,
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
		StructField: "IstioNamespace",
		StringValidation: &cr.StringValidation{
			Default: "istio-system",
		},
	},
	{
		StructField: "Bucket",
		StringValidation: &cr.StringValidation{
			AllowEmpty:       true,
			TreatNullAsEmpty: true,
			Validator:        validateBucketNameOrEmpty,
		},
	},
	{
		StructField: "Telemetry",
		BoolValidation: &cr.BoolValidation{
			Default: true,
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
		StructField: "ImageAsyncGateway",
		StringValidation: &cr.StringValidation{
			Default:   "quay.io/cortexlabs/async-gateway:" + consts.CortexVersion,
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
		StructField: "ImageMetricsServer",
		StringValidation: &cr.StringValidation{
			Default:   "quay.io/cortexlabs/metrics-server:" + consts.CortexVersion,
			Validator: validateImageVersion,
		},
	},
	{
		StructField: "ImageInferentia",
		StringValidation: &cr.StringValidation{
			Default:   "quay.io/cortexlabs/inferentia:" + consts.CortexVersion,
			Validator: validateImageVersion,
		},
	},
	{
		StructField: "ImageNeuronRTD",
		StringValidation: &cr.StringValidation{
			Default:   "quay.io/cortexlabs/neuron-rtd:" + consts.CortexVersion,
			Validator: validateImageVersion,
		},
	},
	{
		StructField: "ImageNvidia",
		StringValidation: &cr.StringValidation{
			Default:   "quay.io/cortexlabs/nvidia:" + consts.CortexVersion,
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
}

var ManagedConfigStructFieldValidations = []*cr.StructFieldValidation{
	{
		StructField: "NodeGroups",
		StructListValidation: &cr.StructListValidation{
			Required: true,
			StructValidation: &cr.StructValidation{
				StructFieldValidations: []*cr.StructFieldValidation{
					{
						StructField: "Name",
						StringValidation: &cr.StringValidation{
							Required:                   true,
							AlphaNumericDashUnderscore: true,
							MaxLength:                  _maxNodeGroupLength,
						},
					},
					{
						StructField: "InstanceType",
						StringValidation: &cr.StringValidation{
							Required:  true,
							MinLength: 1,
							Validator: validateInstanceType,
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
						StructField: "InstanceVolumeSize",
						Int64Validation: &cr.Int64Validation{
							Default:              50,
							GreaterThanOrEqualTo: pointer.Int64(20), // large enough to fit docker images and any other overhead
							LessThanOrEqualTo:    pointer.Int64(16384),
						},
					},
					{
						StructField: "InstanceVolumeType",
						StringValidation: &cr.StringValidation{
							AllowedValues: VolumeTypesStrings(),
							Default:       GP2VolumeType.String(),
						},
						Parser: func(str string) (interface{}, error) {
							return VolumeTypeFromString(str), nil
						},
					},
					{
						StructField: "InstanceVolumeIOPS",
						Int64PtrValidation: &cr.Int64PtrValidation{
							GreaterThanOrEqualTo: pointer.Int64(100),
							LessThanOrEqualTo:    pointer.Int64(64000),
							AllowExplicitNull:    true,
						},
					},
					{
						StructField: "Spot",
						BoolValidation: &cr.BoolValidation{
							Default: false,
						},
					},
					{
						StructField: "SpotConfig",
						StructValidation: &cr.StructValidation{
							DefaultNil:        true,
							AllowExplicitNull: true,
							StructFieldValidations: []*cr.StructFieldValidation{
								{
									StructField: "InstanceDistribution",
									StringListValidation: &cr.StringListValidation{
										DisallowDups:      true,
										Validator:         validateInstanceDistribution,
										AllowExplicitNull: true,
									},
								},
								{
									StructField: "OnDemandBaseCapacity",
									Int64PtrValidation: &cr.Int64PtrValidation{
										GreaterThanOrEqualTo: pointer.Int64(0),
										AllowExplicitNull:    true,
									},
								},
								{
									StructField: "OnDemandPercentageAboveBaseCapacity",
									Int64PtrValidation: &cr.Int64PtrValidation{
										GreaterThanOrEqualTo: pointer.Int64(0),
										LessThanOrEqualTo:    pointer.Int64(100),
										AllowExplicitNull:    true,
									},
								},
								{
									StructField: "MaxPrice",
									Float64PtrValidation: &cr.Float64PtrValidation{
										GreaterThan:       pointer.Float64(0),
										AllowExplicitNull: true,
									},
								},
								{
									StructField: "InstancePools",
									Int64PtrValidation: &cr.Int64PtrValidation{
										GreaterThanOrEqualTo: pointer.Int64(1),
										LessThanOrEqualTo:    pointer.Int64(int64(_maxInstancePools)),
										AllowExplicitNull:    true,
									},
								},
							},
						},
					},
				},
			},
		},
	},
	{
		StructField: "Tags",
		StringMapValidation: &cr.StringMapValidation{
			AllowExplicitNull:  true,
			AllowEmpty:         true,
			ConvertNullToEmpty: true,
			KeyStringValidator: &cr.StringValidation{
				MinLength:                  1,
				MaxLength:                  127,
				DisallowLeadingWhitespace:  true,
				DisallowTrailingWhitespace: true,
				InvalidPrefixes:            _invalidTagPrefixes,
				AWSTag:                     true,
			},
			ValueStringValidator: &cr.StringValidation{
				MinLength:                  1,
				MaxLength:                  255,
				DisallowLeadingWhitespace:  true,
				DisallowTrailingWhitespace: true,
				InvalidPrefixes:            _invalidTagPrefixes,
				AWSTag:                     true,
			},
		},
	},
	{
		StructField: "SSLCertificateARN",
		StringPtrValidation: &cr.StringPtrValidation{
			AllowExplicitNull: true,
		},
	},
	{
		StructField: "CortexPolicyARN",
		StringValidation: &cr.StringValidation{
			Required:         false,
			AllowEmpty:       true,
			TreatNullAsEmpty: true,
		},
	},
	{
		StructField: "IAMPolicyARNs",
		StringListValidation: &cr.StringListValidation{
			Default:           _defaultIAMPolicies,
			AllowEmpty:        true,
			AllowExplicitNull: true,
		},
	},
	{
		StructField: "AvailabilityZones",
		StringListValidation: &cr.StringListValidation{
			AllowEmpty:        true,
			AllowExplicitNull: true,
			DisallowDups:      true,
			InvalidLengths:    []int{1},
		},
	},
	{
		StructField: "SubnetVisibility",
		StringValidation: &cr.StringValidation{
			AllowedValues: SubnetVisibilityStrings(),
			Default:       PublicSubnetVisibility.String(),
		},
		Parser: func(str string) (interface{}, error) {
			return SubnetVisibilityFromString(str), nil
		},
	},
	{
		StructField: "Subnets",
		StructListValidation: &cr.StructListValidation{
			AllowExplicitNull: true,
			MinLength:         2,
			StructValidation: &cr.StructValidation{
				StructFieldValidations: []*cr.StructFieldValidation{
					{
						StructField:      "AvailabilityZone",
						StringValidation: &cr.StringValidation{},
					},
					{
						StructField:      "SubnetID",
						StringValidation: &cr.StringValidation{},
					},
				},
			},
		},
	},
	{
		StructField: "NATGateway",
		StringValidation: &cr.StringValidation{
			AllowedValues: NATGatewayStrings(),
		},
		Parser: func(str string) (interface{}, error) {
			return NATGatewayFromString(str), nil
		},
		DefaultDependentFields: []string{"SubnetVisibility", "Subnets"},
		DefaultDependentFieldsFunc: func(vals []interface{}) interface{} {
			subnetVisibility := vals[0].(SubnetVisibility)
			subnets := vals[1].([]*Subnet)

			if len(subnets) > 0 {
				return NoneNATGateway.String()
			}
			if subnetVisibility == PublicSubnetVisibility {
				return NoneNATGateway.String()
			}
			return SingleNATGateway.String()
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
		StructField: "VPCCIDR",
		StringPtrValidation: &cr.StringPtrValidation{
			Validator: validateVPCCIDR,
		},
	},
}

func CoreConfigValidations(allowExtraFields bool) *cr.StructValidation {
	return &cr.StructValidation{
		Required:               true,
		StructFieldValidations: CoreConfigStructFieldValidations,
		AllowExtraFields:       allowExtraFields,
	}
}

func ManagedConfigValidations(allowExtraFields bool) *cr.StructValidation {
	return &cr.StructValidation{
		Required:               true,
		StructFieldValidations: ManagedConfigStructFieldValidations,
		AllowExtraFields:       allowExtraFields,
	}
}

var FullManagedValidation = &cr.StructValidation{
	Required:               true,
	StructFieldValidations: append([]*cr.StructFieldValidation{}, append(CoreConfigStructFieldValidations, ManagedConfigStructFieldValidations...)...),
}

var AccessValidation = &cr.StructValidation{
	AllowExtraFields: true,
	StructFieldValidations: []*cr.StructFieldValidation{
		{
			StructField: "ClusterName",
			StringValidation: &cr.StringValidation{
				Default:   "cortex",
				MaxLength: 54, // leaves room for 8 char uniqueness string (and "-") for bucket name (63 chars max)
				MinLength: 3,
				Validator: validateClusterName,
			},
		},
		{
			StructField: "Region",
			StringValidation: &cr.StringValidation{
				Required:  true,
				MinLength: 1,
				Validator: RegionValidator,
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

func (cc *Config) ToAccessConfig() AccessConfig {
	return AccessConfig{
		ClusterName:  cc.ClusterName,
		Region:       cc.Region,
		ImageManager: cc.ImageManager,
	}
}

func SQSNamePrefix(clusterName string) string {
	// 8 was chosen to make sure that other identifiers can be added to the full queue name before reaching the 80 char SQS name limit
	return "cx-" + hash.String(clusterName)[:8] + "-"
}

// returns hash of cluster name and adds trailing "-" e.g. cx-abcd1234-
func (cc *CoreConfig) SQSNamePrefix() string {
	return SQSNamePrefix(cc.ClusterName)
}

// this validates the user-provided cluster config
func (cc *Config) Validate(awsClient *aws.Client, skipQuotaVerification bool) error {
	fmt.Print("verifying your configuration ...\n\n")

	numNodeGroups := len(cc.NodeGroups)
	if numNodeGroups == 0 {
		return ErrorNoNodeGroupSpecified()
	}
	if numNodeGroups > MaxNodePoolsOrGroups {
		return ErrorMaxNumOfNodeGroupsReached(MaxNodePoolsOrGroups)
	}

	ngNames := []string{}
	instances := []aws.InstanceTypeRequests{}
	for _, nodeGroup := range cc.NodeGroups {
		if !slices.HasString(ngNames, nodeGroup.Name) {
			ngNames = append(ngNames, nodeGroup.Name)
		} else {
			return errors.Wrap(ErrorDuplicateNodeGroupName(nodeGroup.Name), NodeGroupsKey)
		}

		err := nodeGroup.validateNodeGroup(awsClient, cc.Region)
		if err != nil {
			return errors.Wrap(err, NodeGroupsKey, nodeGroup.Name)
		}

		instances = append(instances, aws.InstanceTypeRequests{
			InstanceType:              nodeGroup.InstanceType,
			RequiredOnDemandInstances: nodeGroup.MaxPossibleOnDemandInstances(),
			RequiredSpotInstances:     nodeGroup.MaxPossibleSpotInstances(),
		})
	}

	if !skipQuotaVerification {
		if err := awsClient.VerifyInstanceQuota(instances); err != nil {
			// Skip AWS errors, since some regions (e.g. eu-north-1) do not support this API
			if !aws.IsAWSError(err) {
				return errors.Wrap(err, NodeGroupsKey)
			}
		}
	}

	if len(cc.AvailabilityZones) > 0 && len(cc.Subnets) > 0 {
		return ErrorSpecifyOneOrNone(AvailabilityZonesKey, SubnetsKey)
	}

	if len(cc.Subnets) > 0 && cc.NATGateway != NoneNATGateway {
		return ErrorNoNATGatewayWithSubnets()
	}

	if cc.SubnetVisibility == PrivateSubnetVisibility && cc.NATGateway == NoneNATGateway && len(cc.Subnets) == 0 {
		return ErrorNATRequiredWithPrivateSubnetVisibility()
	}

	accountID, _, err := awsClient.GetCachedAccountID()
	if err != nil {
		return err
	}

	if cc.Bucket == "" {
		bucketID := hash.String(accountID + cc.Region)[:8] // this is to "guarantee" a globally unique name
		cc.Bucket = cc.ClusterName + "-" + bucketID
	} else {
		bucketRegion, _ := aws.GetBucketRegion(cc.Bucket)
		if bucketRegion != "" && bucketRegion != cc.Region { // if the bucket didn't exist, we will create it in the correct region, so there is no error
			return ErrorS3RegionDiffersFromCluster(cc.Bucket, bucketRegion, cc.Region)
		}
	}

	cc.CortexPolicyARN = DefaultPolicyARN(accountID, cc.ClusterName, cc.Region)

	for _, policyARN := range cc.IAMPolicyARNs {
		_, err := awsClient.IAM().GetPolicy(&iam.GetPolicyInput{
			PolicyArn: pointer.String(policyARN),
		})
		if err != nil {
			if aws.IsErrCode(err, iam.ErrCodeNoSuchEntityException) {
				return errors.Wrap(ErrorIAMPolicyARNNotFound(policyARN), IAMPolicyARNsKey)
			}
			return errors.Wrap(err, IAMPolicyARNsKey)
		}
	}

	if cc.SSLCertificateARN != nil {
		exists, err := awsClient.DoesCertificateExist(*cc.SSLCertificateARN)
		if err != nil {
			return errors.Wrap(err, SSLCertificateARNKey)
		}

		if !exists {
			return errors.Wrap(ErrorSSLCertificateARNNotFound(*cc.SSLCertificateARN, cc.Region), SSLCertificateARNKey)
		}
	}

	for tagName, tagValue := range cc.Tags {
		if strings.HasPrefix(tagName, "cortex.dev/") {
			if tagName != ClusterNameTag {
				return errors.Wrap(cr.ErrorCantHavePrefix(tagName, "cortex.dev/"), TagsKey)
			}
			if tagValue != cc.ClusterName {
				return errors.Wrap(ErrorCantOverrideDefaultTag(), TagsKey)
			}
		}
	}
	cc.Tags[ClusterNameTag] = cc.ClusterName

	if len(cc.Subnets) > 0 {
		if err := cc.validateSubnets(awsClient); err != nil {
			return errors.Wrap(err, SubnetsKey)
		}
	} else {
		if err := cc.setAvailabilityZones(awsClient); err != nil {
			return errors.Wrap(err, AvailabilityZonesKey)
		}
	}

	if !skipQuotaVerification {
		var requiredVPCs int
		if len(cc.Subnets) == 0 {
			requiredVPCs = 1
		}
		if err := awsClient.VerifyNetworkQuotas(1, cc.NATGateway != NoneNATGateway, cc.NATGateway == HighlyAvailableNATGateway, requiredVPCs, strset.FromSlice(cc.AvailabilityZones)); err != nil {
			// Skip AWS errors, since some regions (e.g. eu-north-1) do not support this API
			if !aws.IsAWSError(err) {
				return err
			}
		}
	}

	return nil
}

func (ng *NodeGroup) validateNodeGroup(awsClient *aws.Client, region string) error {
	if ng.MinInstances > ng.MaxInstances {
		return ErrorMinInstancesGreaterThanMax(ng.MinInstances, ng.MaxInstances)
	}

	primaryInstanceType := ng.InstanceType
	if _, ok := aws.InstanceMetadatas[region][primaryInstanceType]; !ok {
		return errors.Wrap(ErrorInstanceTypeNotSupportedInRegion(primaryInstanceType, region), InstanceTypeKey)
	}

	// Throw error if IOPS defined for other storage than io1
	if ng.InstanceVolumeType != IO1VolumeType && ng.InstanceVolumeIOPS != nil {
		return ErrorIOPSNotSupported(ng.InstanceVolumeType)
	}

	if ng.InstanceVolumeType == IO1VolumeType && ng.InstanceVolumeIOPS != nil {
		if *ng.InstanceVolumeIOPS > ng.InstanceVolumeSize*50 {
			return ErrorIOPSTooLarge(*ng.InstanceVolumeIOPS, ng.InstanceVolumeSize)
		}
	}

	if aws.EBSMetadatas[region][ng.InstanceVolumeType.String()].IOPSConfigurable && ng.InstanceVolumeIOPS == nil {
		ng.InstanceVolumeIOPS = pointer.Int64(libmath.MinInt64(ng.InstanceVolumeSize*50, 3000))
	}

	if ng.Spot {
		ng.FillEmptySpotFields(region)

		primaryInstance := aws.InstanceMetadatas[region][primaryInstanceType]

		for _, instanceType := range ng.SpotConfig.InstanceDistribution {
			if instanceType == primaryInstanceType {
				continue
			}
			if _, ok := aws.InstanceMetadatas[region][instanceType]; !ok {
				return errors.Wrap(ErrorInstanceTypeNotSupportedInRegion(instanceType, region), SpotConfigKey, InstanceDistributionKey)
			}

			instanceMetadata := aws.InstanceMetadatas[region][instanceType]
			err := CheckSpotInstanceCompatibility(primaryInstance, instanceMetadata)
			if err != nil {
				return errors.Wrap(err, SpotConfigKey, InstanceDistributionKey)
			}

			spotInstancePrice, awsErr := awsClient.SpotInstancePrice(instanceMetadata.Type)
			if awsErr == nil {
				if err := CheckSpotInstancePriceCompatibility(primaryInstance, instanceMetadata, ng.SpotConfig.MaxPrice, spotInstancePrice); err != nil {
					return errors.Wrap(err, SpotConfigKey, InstanceDistributionKey)
				}
			}
		}

		if ng.SpotConfig.OnDemandBaseCapacity != nil && *ng.SpotConfig.OnDemandBaseCapacity > ng.MaxInstances {
			return ErrorOnDemandBaseCapacityGreaterThanMax(*ng.SpotConfig.OnDemandBaseCapacity, ng.MaxInstances)
		}
	} else {
		if ng.SpotConfig != nil {
			return ErrorConfiguredWhenSpotIsNotEnabled(SpotConfigKey)
		}
	}

	return nil
}

func checkCortexSupport(instanceMetadata aws.InstanceMetadata) error {
	if strings.HasSuffix(instanceMetadata.Type, "nano") ||
		strings.HasSuffix(instanceMetadata.Type, "micro") {
		return ErrorInstanceTypeTooSmall()
	}

	if !aws.IsInstanceSupportedByNLB(instanceMetadata.Type) {
		return ErrorInstanceTypeNotSupported(instanceMetadata.Type)
	}

	if aws.IsARMInstance(instanceMetadata.Type) {
		return ErrorARMInstancesNotSupported(instanceMetadata.Type)
	}

	if err := checkCNISupport(instanceMetadata.Type); err != nil {
		return err
	}

	return nil
}

// Check the instance type against the list of supported instance types for the current default CNI version
// Returns nil if unable to perform the check successfully
func checkCNISupport(instanceType string) error {
	if _cachedCNISupportedInstances == nil {
		// set to empty string to cache errors too
		_cachedCNISupportedInstances = pointer.String("")

		res, err := http.Get(_cniSupportedInstancesURL)
		if err != nil {
			return nil
		}

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return nil
		}
		_ = res.Body.Close()

		_cachedCNISupportedInstances = pointer.String(string(body))
	}

	// sanity check the response
	if !strings.Contains(*_cachedCNISupportedInstances, "m5.large") {
		return nil
	}

	if !strings.Contains(*_cachedCNISupportedInstances, instanceType) {
		return ErrorInstanceTypeNotSupported(instanceType)
	}

	return nil
}

func CheckSpotInstanceCompatibility(target aws.InstanceMetadata, suggested aws.InstanceMetadata) error {
	if target.Inf > 0 && suggested.Inf == 0 {
		return ErrorIncompatibleSpotInstanceTypeInf(suggested)
	}

	if target.GPU > suggested.GPU {
		return ErrorIncompatibleSpotInstanceTypeGPU(target, suggested)
	}

	if target.Memory.Cmp(suggested.Memory) > 0 {
		return ErrorIncompatibleSpotInstanceTypeMemory(target, suggested)
	}

	if target.CPU.Cmp(suggested.CPU) > 0 {
		return ErrorIncompatibleSpotInstanceTypeCPU(target, suggested)
	}

	return nil
}

func CheckSpotInstancePriceCompatibility(target aws.InstanceMetadata, suggested aws.InstanceMetadata, maxPrice *float64, spotInstancePrice float64) error {
	if (maxPrice == nil || *maxPrice == target.Price) && target.Price < spotInstancePrice {
		return ErrorSpotPriceGreaterThanTargetOnDemand(spotInstancePrice, target, suggested)
	}

	if maxPrice != nil && *maxPrice < spotInstancePrice {
		return ErrorSpotPriceGreaterThanMaxPrice(spotInstancePrice, *maxPrice, suggested)
	}
	return nil
}

func AutoGenerateSpotConfig(spotConfig *SpotConfig, region string, instanceType string) {
	primaryInstance := aws.InstanceMetadatas[region][instanceType]
	cleanedDistribution := []string{instanceType}
	for _, spotInstance := range spotConfig.InstanceDistribution {
		if spotInstance != instanceType {
			cleanedDistribution = append(cleanedDistribution, spotInstance)
		}
	}
	spotConfig.InstanceDistribution = cleanedDistribution

	if spotConfig.MaxPrice == nil {
		spotConfig.MaxPrice = &primaryInstance.Price
	}

	if spotConfig.OnDemandBaseCapacity == nil {
		spotConfig.OnDemandBaseCapacity = pointer.Int64(0)
	}

	if spotConfig.OnDemandPercentageAboveBaseCapacity == nil {
		spotConfig.OnDemandPercentageAboveBaseCapacity = pointer.Int64(0)
	}

	if spotConfig.InstancePools == nil {
		if len(spotConfig.InstanceDistribution) < _maxInstancePools {
			spotConfig.InstancePools = pointer.Int64(int64(len(spotConfig.InstanceDistribution)))
		} else {
			spotConfig.InstancePools = pointer.Int64(int64(_maxInstancePools))
		}
	}
}

func (ng *NodeGroup) FillEmptySpotFields(region string) {
	if ng.SpotConfig == nil {
		ng.SpotConfig = &SpotConfig{}
	}
	AutoGenerateSpotConfig(ng.SpotConfig, region, ng.InstanceType)
}

func validateBucketNameOrEmpty(bucket string) (string, error) {
	if bucket == "" {
		return "", nil
	}
	return validateBucketName(bucket)
}

func validateBucketName(bucket string) (string, error) {
	if !_strictS3BucketRegex.MatchString(bucket) {
		return "", errors.Wrap(ErrorDidNotMatchStrictS3Regex(), bucket)
	}
	return bucket, nil
}

func validateVPCCIDR(cidr string) (string, error) {
	_, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", errors.WithStack(err)
	}

	return cidr, nil
}

func validateInstanceType(instanceType string) (string, error) {
	var foundInstance *aws.InstanceMetadata
	for _, instanceMap := range aws.InstanceMetadatas {
		if instanceMetadata, ok := instanceMap[instanceType]; ok {
			foundInstance = &instanceMetadata
			break
		}
	}

	if foundInstance == nil {
		return "", ErrorInvalidInstanceType(instanceType)
	}

	err := checkCortexSupport(*foundInstance)
	if err != nil {
		return "", err
	}

	return instanceType, nil
}

func validateInstanceDistribution(instances []string) ([]string, error) {
	for _, instance := range instances {
		_, err := validateInstanceType(instance)
		if err != nil {
			return nil, err
		}
	}
	return instances, nil
}

// This does not set defaults for fields that are prompted from the user
func SetDefaults(cc *Config) error {
	var emptyMap interface{} = map[interface{}]interface{}{}
	errs := cr.Struct(cc, emptyMap, FullManagedValidation)
	if errors.HasError(errs) {
		return errors.FirstError(errs...)
	}
	return nil
}

// This does not set defaults for fields that are prompted from the user
func GetDefaults() (*Config, error) {
	cc := &Config{}
	err := SetDefaults(cc)
	if err != nil {
		return nil, err
	}

	return cc, nil
}

func (ng *NodeGroup) MaxPossibleOnDemandInstances() int64 {
	if ng.Spot == false || ng.SpotConfig == nil {
		return ng.MaxInstances
	}

	onDemandBaseCap, onDemandPctAboveBaseCap := ng.SpotConfigOnDemandValues()
	return onDemandBaseCap + int64(math.Ceil(float64(onDemandPctAboveBaseCap)/100*float64(ng.MaxInstances-onDemandBaseCap)))
}

func (ng *NodeGroup) MaxPossibleSpotInstances() int64 {
	if ng.Spot == false {
		return 0
	}

	if ng.SpotConfig == nil {
		return ng.MaxInstances
	}

	onDemandBaseCap, onDemandPctAboveBaseCap := ng.SpotConfigOnDemandValues()
	return ng.MaxInstances - onDemandBaseCap - int64(math.Floor(float64(onDemandPctAboveBaseCap)/100*float64(ng.MaxInstances-onDemandBaseCap)))
}

func (ng *NodeGroup) SpotConfigOnDemandValues() (int64, int64) {
	// default OnDemandBaseCapacity is 0
	var onDemandBaseCapacity int64 = 0
	if ng.SpotConfig.OnDemandBaseCapacity != nil {
		onDemandBaseCapacity = *ng.SpotConfig.OnDemandBaseCapacity
	}

	// default OnDemandPercentageAboveBaseCapacity is 0
	var onDemandPercentageAboveBaseCapacity int64 = 0
	if ng.SpotConfig.OnDemandPercentageAboveBaseCapacity != nil {
		onDemandPercentageAboveBaseCapacity = *ng.SpotConfig.OnDemandPercentageAboveBaseCapacity
	}

	return onDemandBaseCapacity, onDemandPercentageAboveBaseCapacity
}

func (cc *CoreConfig) TelemetryEvent() map[string]interface{} {
	event := map[string]interface{}{
		"provider":   types.AWSProviderType,
		"is_managed": cc.IsManaged,
	}

	if cc.ClusterName != "cortex" {
		event["cluster_name._is_custom"] = true
	}

	if cc.Namespace != "default" {
		event["namespace._is_custom"] = true
	}
	if cc.IstioNamespace != "istio-system" {
		event["istio_namespace._is_custom"] = true
	}

	event["region"] = cc.Region

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
	if !strings.HasPrefix(cc.ImageMetricsServer, "cortexlabs/") {
		event["image_metrics_server._is_custom"] = true
	}
	if !strings.HasPrefix(cc.ImageInferentia, "cortexlabs/") {
		event["image_inferentia._is_custom"] = true
	}
	if !strings.HasPrefix(cc.ImageNeuronRTD, "cortexlabs/") {
		event["image_neuron_rtd._is_custom"] = true
	}
	if !strings.HasPrefix(cc.ImageNvidia, "cortexlabs/") {
		event["image_nvidia._is_custom"] = true
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

func (mc *ManagedConfig) TelemetryEvent() map[string]interface{} {
	event := map[string]interface{}{}
	if len(mc.Tags) > 0 {
		event["tags._is_defined"] = true
		event["tags._len"] = len(mc.Tags)
	}
	if len(mc.AvailabilityZones) > 0 {
		event["availability_zones._is_defined"] = true
		event["availability_zones._len"] = len(mc.AvailabilityZones)
		event["availability_zones"] = mc.AvailabilityZones
	}
	if len(mc.Subnets) > 0 {
		event["subnets._is_defined"] = true
		event["subnets._len"] = len(mc.Subnets)
		event["subnets"] = mc.Subnets
	}
	if mc.SSLCertificateARN != nil {
		event["ssl_certificate_arn._is_defined"] = true
	}

	// CortexPolicyARN should be managed by cortex

	if !strset.New(_defaultIAMPolicies...).IsEqual(strset.New(mc.IAMPolicyARNs...)) {
		event["iam_policy_arns._is_custom"] = true
	}
	event["iam_policy_arns._len"] = len(mc.IAMPolicyARNs)

	event["subnet_visibility"] = mc.SubnetVisibility
	event["nat_gateway"] = mc.NATGateway
	event["api_load_balancer_scheme"] = mc.APILoadBalancerScheme
	event["operator_load_balancer_scheme"] = mc.OperatorLoadBalancerScheme
	if mc.VPCCIDR != nil {
		event["vpc_cidr._is_defined"] = true
	}

	onDemandInstanceTypes := strset.New()
	spotInstanceTypes := strset.New()
	var totalMinSize, totalMaxSize int

	event["node_groups._len"] = len(mc.NodeGroups)
	for _, ng := range mc.NodeGroups {
		nodeGroupKey := func(field string) string {
			lifecycle := "on_demand"
			if ng.Spot {
				lifecycle = "spot"
			}
			return fmt.Sprintf("node_groups.%s-%s.%s", ng.InstanceType, lifecycle, field)
		}
		event[nodeGroupKey("_is_defined")] = true
		event[nodeGroupKey("name")] = ng.Name
		event[nodeGroupKey("instance_type")] = ng.InstanceType
		event[nodeGroupKey("min_instances")] = ng.MinInstances
		event[nodeGroupKey("max_instances")] = ng.MaxInstances
		event[nodeGroupKey("instance_volume_size")] = ng.InstanceVolumeSize
		event[nodeGroupKey("instance_volume_type")] = ng.InstanceVolumeType
		if ng.InstanceVolumeIOPS != nil {
			event[nodeGroupKey("instance_volume_iops.is_defined")] = true
			event[nodeGroupKey("instance_volume_iops")] = ng.InstanceVolumeIOPS
		}

		event[nodeGroupKey("spot")] = ng.Spot
		if !ng.Spot {
			onDemandInstanceTypes.Add(ng.InstanceType)
		} else {
			spotInstanceTypes.Add(ng.InstanceType)
		}
		if ng.SpotConfig != nil {
			event[nodeGroupKey("spot_config._is_defined")] = true
			if len(ng.SpotConfig.InstanceDistribution) > 0 {
				event[nodeGroupKey("spot_config.instance_distribution._is_defined")] = true
				event[nodeGroupKey("spot_config.instance_distribution._len")] = len(ng.SpotConfig.InstanceDistribution)
				event[nodeGroupKey("spot_config.instance_distribution")] = ng.SpotConfig.InstanceDistribution
				spotInstanceTypes.Add(ng.SpotConfig.InstanceDistribution...)
			}
			if ng.SpotConfig.OnDemandBaseCapacity != nil {
				event[nodeGroupKey("spot_config.on_demand_base_capacity._is_defined")] = true
				event[nodeGroupKey("spot_config.on_demand_base_capacity")] = *ng.SpotConfig.OnDemandBaseCapacity
			}
			if ng.SpotConfig.OnDemandPercentageAboveBaseCapacity != nil {
				event[nodeGroupKey("spot_config.on_demand_percentage_above_base_capacity._is_defined")] = true
				event[nodeGroupKey("spot_config.on_demand_percentage_above_base_capacity")] = *ng.SpotConfig.OnDemandPercentageAboveBaseCapacity
			}
			if ng.SpotConfig.MaxPrice != nil {
				event[nodeGroupKey("spot_config.max_price._is_defined")] = true
				event[nodeGroupKey("spot_config.max_price")] = *ng.SpotConfig.MaxPrice
			}
			if ng.SpotConfig.InstancePools != nil {
				event[nodeGroupKey("spot_config.instance_pools._is_defined")] = true
				event[nodeGroupKey("spot_config.instance_pools")] = *ng.SpotConfig.InstancePools
			}
		}

		totalMinSize += int(ng.MinInstances)
		totalMaxSize += int(ng.MaxInstances)
	}

	event["node_groups._total_min_size"] = totalMinSize
	event["node_groups._total_max_size"] = totalMaxSize
	event["node_groups._on_demand_instances"] = onDemandInstanceTypes.Slice()
	event["node_groups._spot_instances"] = spotInstanceTypes.Slice()
	event["node_groups._instances"] = strset.Union(onDemandInstanceTypes, spotInstanceTypes).Slice()

	return event
}

func (mc *ManagedConfig) GetAllInstanceTypes() []string {
	allInstanceTypes := strset.New()
	for _, ng := range mc.NodeGroups {
		allInstanceTypes.Add(ng.InstanceType)
	}

	return allInstanceTypes.Slice()
}
