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
	"math"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aws/amazon-vpc-cni-k8s/pkg/awsutils"
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
	"github.com/cortexlabs/yaml"
)

const (
	// MaxNodePoolsOrGroups represents the max number of node groups in a cluster
	MaxNodePoolsOrGroups = 100
	// ClusterNameTag is the tag used for storing a cluster's name in AWS resources
	ClusterNameTag = "cortex.dev/cluster-name"
	// SQSQueueDelimiter is the delimiter character used for naming cortex SQS queues (e.g. cx_<cluster_hash>_b_<api_name>_<jon_id>)
	// In this case, _ was chosen to simplify the retrieval of information for the queue's name,
	// since the api naming scheme does not allow this character.
	SQSQueueDelimiter = "_"
)

var (
	_maxNodeGroupLengthWithPrefix = 32
	_maxNodeGroupLength           = _maxNodeGroupLengthWithPrefix - len("cx-wd-") // or cx-ws-
	_maxInstancePools             = 20
	_defaultIAMPolicies           = []string{"arn:aws:iam::aws:policy/AmazonS3FullAccess"}
	_invalidTagPrefixes           = []string{"kubernetes.io/", "k8s.io/", "eksctl.", "alpha.eksctl.", "beta.eksctl.", "aws:", "Aws:", "aWs:", "awS:", "aWS:", "AwS:", "aWS:", "AWS:"}

	_smallestIOPSForIO1VolumeType = int64(100)
	_highestIOPSForIO1VolumeType  = int64(64000)
	_smallestIOPSForGP3VolumeType = int64(3000)
	_highestIOPSForGP3VolumeType  = int64(16000)

	_maxIOPSToVolumeSizeRatioForIO1 = int64(50)
	_maxIOPSToVolumeSizeRatioForGP3 = int64(500)
	_minIOPSToThroughputRatioForGP3 = int64(4)

	// This regex is stricter than the actual S3 rules
	_strictS3BucketRegex = regexp.MustCompile(`^([a-z0-9])+(-[a-z0-9]+)*$`)
)

type CoreConfig struct {
	// Non-user-specifiable fields
	ClusterUID     string `json:"cluster_uid" yaml:"cluster_uid"`
	Bucket         string `json:"bucket" yaml:"bucket"`
	Telemetry      bool   `json:"telemetry" yaml:"telemetry"`
	Namespace      string `json:"namespace" yaml:"namespace"`
	IstioNamespace string `json:"istio_namespace" yaml:"istio_namespace"`

	// User-specifiable fields
	ClusterName string `json:"cluster_name" yaml:"cluster_name"`
	Region      string `json:"region" yaml:"region"`

	// User-specifiable fields
	ImageOperator                   string `json:"image_operator" yaml:"image_operator"`
	ImageControllerManager          string `json:"image_controller_manager" yaml:"image_controller_manager"`
	ImageManager                    string `json:"image_manager" yaml:"image_manager"`
	ImageKubexit                    string `json:"image_kubexit" yaml:"image_kubexit"`
	ImageProxy                      string `json:"image_proxy" yaml:"image_proxy"`
	ImageAsyncGateway               string `json:"image_async_gateway" yaml:"image_async_gateway"`
	ImageEnqueuer                   string `json:"image_enqueuer" yaml:"image_enqueuer"`
	ImageDequeuer                   string `json:"image_dequeuer" yaml:"image_dequeuer"`
	ImageClusterAutoscaler          string `json:"image_cluster_autoscaler" yaml:"image_cluster_autoscaler"`
	ImageMetricsServer              string `json:"image_metrics_server" yaml:"image_metrics_server"`
	ImageInferentia                 string `json:"image_inferentia" yaml:"image_inferentia"`
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
	NodeGroups                        []*NodeGroup       `json:"node_groups" yaml:"node_groups"`
	Tags                              map[string]string  `json:"tags" yaml:"tags"`
	AvailabilityZones                 []string           `json:"availability_zones" yaml:"availability_zones"`
	SSLCertificateARN                 *string            `json:"ssl_certificate_arn,omitempty" yaml:"ssl_certificate_arn,omitempty"`
	IAMPolicyARNs                     []string           `json:"iam_policy_arns" yaml:"iam_policy_arns"`
	SubnetVisibility                  SubnetVisibility   `json:"subnet_visibility" yaml:"subnet_visibility"`
	Subnets                           []*Subnet          `json:"subnets,omitempty" yaml:"subnets,omitempty"`
	NATGateway                        NATGateway         `json:"nat_gateway" yaml:"nat_gateway"`
	APILoadBalancerScheme             LoadBalancerScheme `json:"api_load_balancer_scheme" yaml:"api_load_balancer_scheme"`
	OperatorLoadBalancerScheme        LoadBalancerScheme `json:"operator_load_balancer_scheme" yaml:"operator_load_balancer_scheme"`
	APILoadBalancerCIDRWhiteList      []string           `json:"api_load_balancer_cidr_white_list,omitempty" yaml:"api_load_balancer_cidr_white_list,omitempty"`
	OperatorLoadBalancerCIDRWhiteList []string           `json:"operator_load_balancer_cidr_white_list,omitempty" yaml:"operator_load_balancer_cidr_white_list,omitempty"`
	VPCCIDR                           *string            `json:"vpc_cidr,omitempty" yaml:"vpc_cidr,omitempty"`
	CortexPolicyARN                   string             `json:"cortex_policy_arn" yaml:"cortex_policy_arn"` // this field is not user facing
	AccountID                         string             `json:"account_id" yaml:"account_id"`               // this field is not user facing
}

type NodeGroup struct {
	Name                     string      `json:"name" yaml:"name"`
	InstanceType             string      `json:"instance_type" yaml:"instance_type"`
	MinInstances             int64       `json:"min_instances" yaml:"min_instances"`
	MaxInstances             int64       `json:"max_instances" yaml:"max_instances"`
	Priority                 int64       `json:"priority" yaml:"priority"`
	InstanceVolumeSize       int64       `json:"instance_volume_size" yaml:"instance_volume_size"`
	InstanceVolumeType       VolumeType  `json:"instance_volume_type" yaml:"instance_volume_type"`
	InstanceVolumeIOPS       *int64      `json:"instance_volume_iops" yaml:"instance_volume_iops"`
	InstanceVolumeThroughput *int64      `json:"instance_volume_throughput" yaml:"instance_volume_throughput"`
	Spot                     bool        `json:"spot" yaml:"spot"`
	SpotConfig               *SpotConfig `json:"spot_config" yaml:"spot_config"`
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

// NewForFile initializes and validates the cluster config from the YAML config file
func NewForFile(clusterConfigPath string) (*Config, error) {
	coreConfig := CoreConfig{}
	errs := cr.ParseYAMLFile(&coreConfig, CoreConfigValidations(true), clusterConfigPath)
	if errors.HasError(errs) {
		return nil, errors.FirstError(errs...)
	}

	managedConfig := ManagedConfig{}
	errs = cr.ParseYAMLFile(&managedConfig, ManagedConfigValidations(true), clusterConfigPath)
	if errors.HasError(errs) {
		return nil, errors.FirstError(errs...)
	}

	return &Config{
		CoreConfig:    coreConfig,
		ManagedConfig: managedConfig,
	}, nil
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

	configHash := sha256.New()
	configHash.Write(bytes)
	return hex.EncodeToString(configHash.Sum(nil)), nil
}

var CoreConfigStructFieldValidations = []*cr.StructFieldValidation{
	{
		Key: "provider",
		StringValidation: &cr.StringValidation{
			AllowEmpty: true,
			Validator: func(provider string) (string, error) {
				if provider == "" || provider == "aws" {
					return "", nil
				}
				if provider == "gcp" || provider == "local" {
					return "", ErrorInvalidLegacyProvider(provider)
				}
				return "", ErrorInvalidProvider(provider)
			},
		},
	},
	{
		StructField: "ClusterUID",
		StringValidation: &cr.StringValidation{
			Default:          "",
			AllowEmpty:       true,
			TreatNullAsEmpty: true,
		},
	},
	{
		StructField: "Bucket",
		StringValidation: &cr.StringValidation{
			Default:          "",
			AllowEmpty:       true,
			TreatNullAsEmpty: true,
		},
	},
	{
		StructField: "Namespace",
		StringValidation: &cr.StringValidation{
			Default:       "default",
			AllowedValues: []string{"default"},
		},
	},
	{
		StructField: "IstioNamespace",
		StringValidation: &cr.StringValidation{
			Default:       "istio-system",
			AllowedValues: []string{"istio-system"},
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
		StructField: "Telemetry",
		BoolValidation: &cr.BoolValidation{
			Default: true,
		},
	},
	{
		StructField: "ImageOperator",
		StringValidation: &cr.StringValidation{
			Default:   consts.DefaultRegistry() + "/operator:" + consts.CortexVersion,
			Validator: validateImageVersion,
		},
	},
	{
		StructField: "ImageControllerManager",
		StringValidation: &cr.StringValidation{
			Default:   consts.DefaultRegistry() + "/controller-manager:" + consts.CortexVersion,
			Validator: validateImageVersion,
		},
	},
	{
		StructField: "ImageManager",
		StringValidation: &cr.StringValidation{
			Default:   consts.DefaultRegistry() + "/manager:" + consts.CortexVersion,
			Validator: validateImageVersion,
		},
	},
	{
		StructField: "ImageKubexit",
		StringValidation: &cr.StringValidation{
			Default:   consts.DefaultRegistry() + "/kubexit:" + consts.CortexVersion,
			Validator: validateImageVersion,
		},
	},
	{
		StructField: "ImageProxy",
		StringValidation: &cr.StringValidation{
			Default:   consts.DefaultRegistry() + "/proxy:" + consts.CortexVersion,
			Validator: validateImageVersion,
		},
	},
	{
		StructField: "ImageAsyncGateway",
		StringValidation: &cr.StringValidation{
			Default:   consts.DefaultRegistry() + "/async-gateway:" + consts.CortexVersion,
			Validator: validateImageVersion,
		},
	},
	{
		StructField: "ImageEnqueuer",
		StringValidation: &cr.StringValidation{
			Default:   consts.DefaultRegistry() + "/enqueuer:" + consts.CortexVersion,
			Validator: validateImageVersion,
		},
	},
	{
		StructField: "ImageDequeuer",
		StringValidation: &cr.StringValidation{
			Default:   consts.DefaultRegistry() + "/dequeuer:" + consts.CortexVersion,
			Validator: validateImageVersion,
		},
	},
	{
		StructField: "ImageClusterAutoscaler",
		StringValidation: &cr.StringValidation{
			Default:   consts.DefaultRegistry() + "/cluster-autoscaler:" + consts.CortexVersion,
			Validator: validateImageVersion,
		},
	},
	{
		StructField: "ImageMetricsServer",
		StringValidation: &cr.StringValidation{
			Default:   consts.DefaultRegistry() + "/metrics-server:" + consts.CortexVersion,
			Validator: validateImageVersion,
		},
	},
	{
		StructField: "ImageInferentia",
		StringValidation: &cr.StringValidation{
			Default:   consts.DefaultRegistry() + "/inferentia:" + consts.CortexVersion,
			Validator: validateImageVersion,
		},
	},
	{
		StructField: "ImageNvidia",
		StringValidation: &cr.StringValidation{
			Default:   consts.DefaultRegistry() + "/nvidia:" + consts.CortexVersion,
			Validator: validateImageVersion,
		},
	},
	{
		StructField: "ImageFluentBit",
		StringValidation: &cr.StringValidation{
			Default:   consts.DefaultRegistry() + "/fluent-bit:" + consts.CortexVersion,
			Validator: validateImageVersion,
		},
	},
	{
		StructField: "ImageIstioProxy",
		StringValidation: &cr.StringValidation{
			Default:   consts.DefaultRegistry() + "/istio-proxy:" + consts.CortexVersion,
			Validator: validateImageVersion,
		},
	},
	{
		StructField: "ImageIstioPilot",
		StringValidation: &cr.StringValidation{
			Default:   consts.DefaultRegistry() + "/istio-pilot:" + consts.CortexVersion,
			Validator: validateImageVersion,
		},
	},
	{
		StructField: "ImagePrometheus",
		StringValidation: &cr.StringValidation{
			Default:   consts.DefaultRegistry() + "/prometheus:" + consts.CortexVersion,
			Validator: validateImageVersion,
		},
	},
	{
		StructField: "ImagePrometheusConfigReloader",
		StringValidation: &cr.StringValidation{
			Default:   consts.DefaultRegistry() + "/prometheus-config-reloader:" + consts.CortexVersion,
			Validator: validateImageVersion,
		},
	},
	{
		StructField: "ImagePrometheusOperator",
		StringValidation: &cr.StringValidation{
			Default:   consts.DefaultRegistry() + "/prometheus-operator:" + consts.CortexVersion,
			Validator: validateImageVersion,
		},
	},
	{
		StructField: "ImagePrometheusStatsDExporter",
		StringValidation: &cr.StringValidation{
			Default:   consts.DefaultRegistry() + "/prometheus-statsd-exporter:" + consts.CortexVersion,
			Validator: validateImageVersion,
		},
	},
	{
		StructField: "ImagePrometheusDCGMExporter",
		StringValidation: &cr.StringValidation{
			Default:   consts.DefaultRegistry() + "/prometheus-dcgm-exporter:" + consts.CortexVersion,
			Validator: validateImageVersion,
		},
	},
	{
		StructField: "ImagePrometheusKubeStateMetrics",
		StringValidation: &cr.StringValidation{
			Default:   consts.DefaultRegistry() + "/prometheus-kube-state-metrics:" + consts.CortexVersion,
			Validator: validateImageVersion,
		},
	},
	{
		StructField: "ImagePrometheusNodeExporter",
		StringValidation: &cr.StringValidation{
			Default:   consts.DefaultRegistry() + "/prometheus-node-exporter:" + consts.CortexVersion,
			Validator: validateImageVersion,
		},
	},
	{
		StructField: "ImageKubeRBACProxy",
		StringValidation: &cr.StringValidation{
			Default:   consts.DefaultRegistry() + "/kube-rbac-proxy:" + consts.CortexVersion,
			Validator: validateImageVersion,
		},
	},
	{
		StructField: "ImageGrafana",
		StringValidation: &cr.StringValidation{
			Default:   consts.DefaultRegistry() + "/grafana:" + consts.CortexVersion,
			Validator: validateImageVersion,
		},
	},
	{
		StructField: "ImageEventExporter",
		StringValidation: &cr.StringValidation{
			Default:   consts.DefaultRegistry() + "/event-exporter:" + consts.CortexVersion,
			Validator: validateImageVersion,
		},
	},
}

var ManagedConfigStructFieldValidations = []*cr.StructFieldValidation{
	{
		StructField: "NodeGroups",
		StructListValidation: &cr.StructListValidation{
			Required:         true,
			StructValidation: nodeGroupsFieldValidation(),
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
		StructField: "APILoadBalancerCIDRWhiteList",
		StringListValidation: &cr.StringListValidation{
			Validator: func(addresses []string) ([]string, error) {
				for i, address := range addresses {
					_, err := validateCIDR(address)
					if err != nil {
						return nil, errors.Wrap(err, fmt.Sprintf("index %d", i))
					}
				}
				return addresses, nil
			},
		},
	},
	{
		StructField: "OperatorLoadBalancerCIDRWhiteList",
		StringListValidation: &cr.StringListValidation{
			Validator: func(addresses []string) ([]string, error) {
				for i, address := range addresses {
					_, err := validateCIDR(address)
					if err != nil {
						return nil, errors.Wrap(err, fmt.Sprintf("index %d", i))
					}
				}
				return addresses, nil
			},
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
			Validator: validateCIDR,
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
		StructField: "AccountID",
		StringValidation: &cr.StringValidation{
			Required:         false,
			AllowEmpty:       true,
			TreatNullAsEmpty: true,
		},
	},
}

func nodeGroupsFieldValidation() *cr.StructValidation {
	return &cr.StructValidation{
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
					Default:              int64(5),
					GreaterThanOrEqualTo: pointer.Int64(0), // this will be validated to be > 0 during cluster up (can be scaled down later)
				},
			},
			{
				StructField: "Priority",
				Int64Validation: &cr.Int64Validation{
					Default:              int64(0),
					GreaterThanOrEqualTo: pointer.Int64(0),
					LessThanOrEqualTo:    pointer.Int64(100),
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
					Default:       GP3VolumeType.String(),
				},
				Parser: func(str string) (interface{}, error) {
					return VolumeTypeFromString(str), nil
				},
			},
			{
				StructField: "InstanceVolumeIOPS",
				Int64PtrValidation: &cr.Int64PtrValidation{
					AllowExplicitNull: true,
				},
			},
			{
				StructField: "InstanceVolumeThroughput",
				Int64PtrValidation: &cr.Int64PtrValidation{
					GreaterThanOrEqualTo: pointer.Int64(125),
					LessThanOrEqualTo:    pointer.Int64(1000),
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
	}
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
				Default:   consts.DefaultRegistry() + "/manager:" + consts.CortexVersion,
				Validator: validateImageVersion,
			},
		},
	},
}

var NodeGroupsValidation = nodeGroupsFieldValidation()

func (cc *Config) ToAccessConfig() AccessConfig {
	return AccessConfig{
		ClusterName:  cc.ClusterName,
		Region:       cc.Region,
		ImageManager: cc.ImageManager,
	}
}

func SQSNamePrefix(clusterName string) string {
	// 8 was chosen to make sure that other identifiers can be added to the full queue name before reaching the 80 char SQS name limit
	return "cx" + SQSQueueDelimiter + hash.String(clusterName)[:8] + SQSQueueDelimiter
}

// SQSNamePrefix returns a string with the hash of cluster name and adds trailing "_" e.g. cx_abcd1234_
func (cc *CoreConfig) SQSNamePrefix() string {
	return SQSNamePrefix(cc.ClusterName)
}

// this validates the user-provided cluster config
func (cc *Config) ValidateOnInstall(awsClient *aws.Client) error {
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
		// setting max_instances to 0 during cluster creation is not permitted (but scaling max_instances to 0 afterwards is allowed)
		if nodeGroup.MaxInstances == 0 {
			return errors.Wrap(ErrorNodeGroupMaxInstancesIsZero(), NodeGroupsKey, nodeGroup.Name)
		}
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

	if err := awsClient.VerifyInstanceQuota(instances); err != nil {
		// Skip AWS errors, since some regions (e.g. eu-north-1) do not support this API
		if !aws.IsAWSError(err) {
			return errors.Wrap(err, NodeGroupsKey)
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

	if cc.AccountID != "" {
		return ErrorDisallowedField(AccountIDKey)
	}
	cc.AccountID = accountID

	if cc.Bucket != "" {
		return ErrorDisallowedField(BucketKey)
	}
	cc.Bucket = BucketName(accountID, cc.ClusterName, cc.Region)
	// check if the bucket already exists in a different region for some reason
	bucketRegion, _ := aws.GetBucketRegion(cc.Bucket)
	if bucketRegion != "" && bucketRegion != cc.Region { // if the bucket didn't exist, we will create it in the correct region, so there is no error
		return ErrorS3RegionDiffersFromCluster(cc.Bucket, bucketRegion, cc.Region)
	}

	if cc.ClusterUID != "" {
		return ErrorDisallowedField(ClusterUIDKey)
	}
	cc.ClusterUID = strconv.FormatInt(time.Now().Unix(), 10)

	if cc.CortexPolicyARN != "" {
		return ErrorDisallowedField(CortexPolicyARNKey)
	}
	cc.CortexPolicyARN = DefaultPolicyARN(accountID, cc.ClusterName, cc.Region)

	defaultPoliciesSet := strset.New(_defaultIAMPolicies...)
	for i := range cc.IAMPolicyARNs {
		policyARN := cc.IAMPolicyARNs[i]

		if defaultPoliciesSet.Has(policyARN) {
			partition := aws.PartitionFromRegion(cc.Region)
			adjustedPolicyARN := strings.Replace(policyARN, "arn:aws:", fmt.Sprintf("arn:%s:", partition), 1)
			cc.IAMPolicyARNs[i] = adjustedPolicyARN
			policyARN = adjustedPolicyARN
		}
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

	var requiredVPCs int
	if len(cc.Subnets) == 0 {
		requiredVPCs = 1
	}
	longestCIDRWhiteList := libmath.MaxInt(len(cc.APILoadBalancerCIDRWhiteList), len(cc.OperatorLoadBalancerCIDRWhiteList))
	if err := awsClient.VerifyNetworkQuotas(1, cc.NATGateway != NoneNATGateway, cc.NATGateway == HighlyAvailableNATGateway, requiredVPCs, strset.FromSlice(cc.AvailabilityZones), len(cc.NodeGroups), longestCIDRWhiteList, false); err != nil {
		// Skip AWS errors, since some regions (e.g. eu-north-1) do not support this API
		if !aws.IsAWSError(err) {
			return err
		}
	}

	return nil
}

func (cc *Config) ValidateOnConfigure(awsClient *aws.Client, oldConfig Config) (newNgNames, removedNgNames, scaledNgNames []string, errReturned error) {
	fmt.Print("verifying your configuration ...\n\n")

	// fill-out empty fields
	cc.ClusterUID = oldConfig.ClusterUID
	cc.Bucket = oldConfig.Bucket
	cc.ClusterName = oldConfig.ClusterName
	cc.Region = oldConfig.Region
	if len(cc.Tags) == 0 {
		cc.Tags = oldConfig.Tags
	}
	if len(cc.AvailabilityZones) == 0 {
		cc.AvailabilityZones = oldConfig.AvailabilityZones
	}
	if cc.SSLCertificateARN == nil && oldConfig.SSLCertificateARN != nil {
		cc.SSLCertificateARN = pointer.String(*oldConfig.SSLCertificateARN)
	}
	if len(cc.IAMPolicyARNs) == 0 {
		cc.IAMPolicyARNs = oldConfig.IAMPolicyARNs
	}
	if len(cc.Subnets) == 0 {
		for _, subnet := range oldConfig.Subnets {
			if subnet == nil {
				continue
			}
			auxSubnet := *subnet
			cc.Subnets = append(cc.Subnets, &auxSubnet)
		}
	}
	if len(cc.APILoadBalancerCIDRWhiteList) == 0 {
		cc.APILoadBalancerCIDRWhiteList = oldConfig.APILoadBalancerCIDRWhiteList
	}
	if len(cc.OperatorLoadBalancerCIDRWhiteList) == 0 {
		cc.OperatorLoadBalancerCIDRWhiteList = oldConfig.OperatorLoadBalancerCIDRWhiteList
	}
	if cc.VPCCIDR == nil && oldConfig.VPCCIDR != nil {
		cc.VPCCIDR = pointer.String(*oldConfig.VPCCIDR)
	}
	if cc.CortexPolicyARN == "" {
		cc.CortexPolicyARN = oldConfig.CortexPolicyARN
	}
	if cc.AccountID == "" {
		cc.AccountID = oldConfig.AccountID
	}

	// validate actionable changes
	newClusterConfigCopy, err := cc.DeepCopy()
	if err != nil {
		errReturned = err
		return
	}

	oldClusterConfigCopy, err := oldConfig.DeepCopy()
	if err != nil {
		errReturned = err
		return
	}

	newClusterConfigCopy.NodeGroups = []*NodeGroup{}
	oldClusterConfigCopy.NodeGroups = []*NodeGroup{}

	h1, err := newClusterConfigCopy.Hash()
	if err != nil {
		errReturned = err
		return
	}
	h2, err := oldClusterConfigCopy.Hash()
	if err != nil {
		errReturned = err
		return
	}
	if h1 != h2 {
		errReturned = ErrorConfigCannotBeChangedOnUpdate()
		return
	}

	numNodeGroups := len(cc.NodeGroups)
	if numNodeGroups == 0 {
		errReturned = ErrorNoNodeGroupSpecified()
		return
	}
	if numNodeGroups > MaxNodePoolsOrGroups {
		errReturned = ErrorMaxNumOfNodeGroupsReached(MaxNodePoolsOrGroups)
		return
	}

	// validate the nodegroups
	instances := []aws.InstanceTypeRequests{}
	ngNames := []string{}
	for _, ng := range cc.NodeGroups {
		// setting max_instances to 0 during node group creation is not permitted (but scaling max_instances to 0 afterwards is allowed)
		if ng.MaxInstances == 0 {
			errReturned = errors.Wrap(ErrorNodeGroupMaxInstancesIsZero(), NodeGroupsKey, ng.Name)
			return
		}
		if !slices.HasString(ngNames, ng.Name) {
			ngNames = append(ngNames, ng.Name)
		} else {
			errReturned = errors.Wrap(ErrorDuplicateNodeGroupName(ng.Name), NodeGroupsKey)
			return
		}

		err := ng.validateNodeGroup(awsClient, cc.Region)
		if err != nil {
			errReturned = errors.Wrap(err, NodeGroupsKey, ng.Name)
			return
		}

		instances = append(instances, aws.InstanceTypeRequests{
			InstanceType:              ng.InstanceType,
			RequiredOnDemandInstances: ng.MaxPossibleOnDemandInstances(),
			RequiredSpotInstances:     ng.MaxPossibleSpotInstances(),
		})
	}

	if err := awsClient.VerifyInstanceQuota(instances); err != nil {
		// Skip AWS errors, since some regions (e.g. eu-north-1) do not support this API
		if !aws.IsAWSError(err) {
			errReturned = errors.Wrap(err, NodeGroupsKey)
			return
		}
	}

	retainedNewNgs, retainedOldNgs := cc.getCommonNodeGroups(oldConfig)
	newNgs := cc.getNewNodeGroups(oldConfig)
	removedNgs := cc.getRemovedNodeGroups(oldConfig)

	netAdditionOfNgs := len(newNgs) - len(removedNgs)
	if netAdditionOfNgs > 0 {
		longestCIDRWhiteList := libmath.MaxInt(len(cc.APILoadBalancerCIDRWhiteList), len(cc.OperatorLoadBalancerCIDRWhiteList))
		if err := awsClient.VerifyNetworkQuotasOnNodeGroupsAddition(strset.FromSlice(cc.AvailabilityZones), netAdditionOfNgs, longestCIDRWhiteList); err != nil {
			// Skip AWS errors, since some regions (e.g. eu-north-1) do not support this API
			if !aws.IsAWSError(err) {
				errReturned = errors.Wrap(err, NodeGroupsKey)
				return
			}
		}
	}

	// validate existing nodegroups
	scaledNgs := []*NodeGroup{}
	for i := range retainedNewNgs {
		newNgCopy, err := retainedNewNgs[i].DeepCopy()
		if err != nil {
			errReturned = errors.Wrap(err, NodeGroupsKey)
			return
		}
		oldNgCopy, err := retainedOldNgs[i].DeepCopy()
		if err != nil {
			errReturned = errors.Wrap(err, NodeGroupsKey)
			return
		}

		newNgCopy.MinInstances = 0
		newNgCopy.MaxInstances = 0
		oldNgCopy.MinInstances = 0
		oldNgCopy.MaxInstances = 0

		newHash, err := newNgCopy.Hash()
		if err != nil {
			errReturned = errors.Wrap(err, NodeGroupsKey)
			return
		}
		oldHash, err := oldNgCopy.Hash()
		if err != nil {
			errReturned = errors.Wrap(err, NodeGroupsKey)
			return
		}

		if newHash != oldHash {
			errReturned = errors.Wrap(ErrorNodeGroupCanOnlyBeScaled(), NodeGroupsKey, newNgCopy.Name)
			return
		}

		if retainedNewNgs[i].MinInstances != retainedOldNgs[i].MinInstances || retainedNewNgs[i].MaxInstances != retainedOldNgs[i].MaxInstances {
			scaledNgs = append(scaledNgs, retainedNewNgs[i])
		}
	}

	scaledNgNames = GetNodeGroupNames(scaledNgs)
	newNgNames = GetNodeGroupNames(newNgs)
	removedNgNames = GetNodeGroupNames(removedNgs)
	return
}

func (ng *NodeGroup) validateNodeGroup(awsClient *aws.Client, region string) error {
	if ng.MinInstances > ng.MaxInstances {
		return ErrorMinInstancesGreaterThanMax(ng.MinInstances, ng.MaxInstances)
	}

	primaryInstanceType := ng.InstanceType

	if !aws.InstanceTypes[region].Has(primaryInstanceType) {
		return errors.Wrap(ErrorInstanceTypeNotSupportedInRegion(primaryInstanceType, region), InstanceTypeKey)
	}

	if _, ok := aws.InstanceMetadatas[region][primaryInstanceType]; !ok {
		return errors.Wrap(ErrorInstanceTypeNotSupportedByCortex(primaryInstanceType), InstanceTypeKey)
	}

	// throw error if IOPS defined for other storage than io1/gp3
	if ng.InstanceVolumeType != IO1VolumeType && ng.InstanceVolumeType != GP3VolumeType && ng.InstanceVolumeIOPS != nil {
		return ErrorIOPSNotSupported(ng.InstanceVolumeType)
	}

	// throw error if throughput defined for other storage than gp3
	if ng.InstanceVolumeType != GP3VolumeType && ng.InstanceVolumeThroughput != nil {
		return ErrorThroughputNotSupported(ng.InstanceVolumeType)
	}

	if ng.InstanceVolumeType == GP3VolumeType && ((ng.InstanceVolumeIOPS != nil && ng.InstanceVolumeThroughput == nil) || (ng.InstanceVolumeIOPS == nil && ng.InstanceVolumeThroughput != nil)) {
		return ErrorSpecifyTwoOrNone(InstanceVolumeIOPSKey, InstanceVolumeThroughputKey)
	}

	if ng.InstanceVolumeIOPS != nil {
		if ng.InstanceVolumeType == IO1VolumeType {
			if *ng.InstanceVolumeIOPS < _smallestIOPSForIO1VolumeType {
				return ErrorIOPSTooSmall(ng.InstanceVolumeType, *ng.InstanceVolumeIOPS, _smallestIOPSForIO1VolumeType)
			}
			if *ng.InstanceVolumeIOPS > _highestIOPSForIO1VolumeType {
				return ErrorIOPSTooLarge(ng.InstanceVolumeType, *ng.InstanceVolumeIOPS, _highestIOPSForIO1VolumeType)
			}
			if *ng.InstanceVolumeIOPS > ng.InstanceVolumeSize*_maxIOPSToVolumeSizeRatioForIO1 {
				return ErrorIOPSToVolumeSizeRatio(ng.InstanceVolumeType, _maxIOPSToVolumeSizeRatioForIO1, *ng.InstanceVolumeIOPS, ng.InstanceVolumeSize)
			}
		} else {
			if *ng.InstanceVolumeIOPS < _smallestIOPSForGP3VolumeType {
				return ErrorIOPSTooSmall(ng.InstanceVolumeType, *ng.InstanceVolumeIOPS, _smallestIOPSForGP3VolumeType)
			}
			if *ng.InstanceVolumeIOPS > _highestIOPSForGP3VolumeType {
				return ErrorIOPSTooLarge(ng.InstanceVolumeType, *ng.InstanceVolumeIOPS, _highestIOPSForGP3VolumeType)
			}
			if *ng.InstanceVolumeIOPS > ng.InstanceVolumeSize*_maxIOPSToVolumeSizeRatioForGP3 {
				return ErrorIOPSToVolumeSizeRatio(ng.InstanceVolumeType, _maxIOPSToVolumeSizeRatioForGP3, *ng.InstanceVolumeIOPS, ng.InstanceVolumeSize)
			}
			iopsToThroughputRatio := float64(*ng.InstanceVolumeIOPS) / float64(*ng.InstanceVolumeThroughput)
			if iopsToThroughputRatio < float64(_minIOPSToThroughputRatioForGP3) {
				return ErrorIOPSToThroughputRatio(ng.InstanceVolumeType, _minIOPSToThroughputRatioForGP3, *ng.InstanceVolumeIOPS, *ng.InstanceVolumeThroughput)
			}
		}
	} else if ng.InstanceVolumeType == GP3VolumeType {
		ng.InstanceVolumeIOPS = pointer.Int64(3000)
		ng.InstanceVolumeThroughput = pointer.Int64(125)
	} else if ng.InstanceVolumeType == IO1VolumeType {
		ng.InstanceVolumeIOPS = pointer.Int64(libmath.MinInt64(ng.InstanceVolumeSize*_maxIOPSToVolumeSizeRatioForIO1, 3000))
	}

	if ng.Spot {
		ng.FillEmptySpotFields(region)

		primaryInstance := aws.InstanceMetadatas[region][primaryInstanceType]

		for _, instanceType := range ng.SpotConfig.InstanceDistribution {
			if instanceType == primaryInstanceType {
				continue
			}

			if !aws.InstanceTypes[region].Has(instanceType) {
				return errors.Wrap(ErrorInstanceTypeNotSupportedInRegion(instanceType, region), SpotConfigKey, InstanceDistributionKey)
			}

			if _, ok := aws.InstanceMetadatas[region][instanceType]; !ok {
				return errors.Wrap(ErrorInstanceTypeNotSupportedByCortex(instanceType), SpotConfigKey, InstanceDistributionKey)
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

func (cc *Config) GetNodeGroupByName(name string) *NodeGroup {
	for _, ng := range cc.NodeGroups {
		if ng.Name == name {
			return ng
		}
	}
	return nil
}

func (cc *Config) getNewNodeGroups(oldConfig Config) []*NodeGroup {
	newNodeGroups := []*NodeGroup{}
	for _, updatingNg := range cc.NodeGroups {
		isNewNg := true
		for _, previousNg := range oldConfig.NodeGroups {
			if previousNg.Name == updatingNg.Name {
				isNewNg = false
				break
			}
		}
		if isNewNg {
			ngCopy := *updatingNg
			newNodeGroups = append(newNodeGroups, &ngCopy)
		}
	}
	return newNodeGroups
}

func (cc *Config) getRemovedNodeGroups(oldConfig Config) []*NodeGroup {
	removedNodeGroups := []*NodeGroup{}
	for _, previousNg := range oldConfig.NodeGroups {
		isRemovedNg := true
		for _, updatingNg := range cc.NodeGroups {
			if previousNg.Name == updatingNg.Name {
				isRemovedNg = false
				break
			}
		}
		if isRemovedNg {
			ngCopy := *previousNg
			removedNodeGroups = append(removedNodeGroups, &ngCopy)
		}
	}
	return removedNodeGroups
}

func (cc *Config) getCommonNodeGroups(oldConfig Config) ([]*NodeGroup, []*NodeGroup) {
	commonNewNodeGroups := []*NodeGroup{}
	commonOldNodeGroups := []*NodeGroup{}
	for _, previousNg := range oldConfig.NodeGroups {
		for _, updatingNg := range cc.NodeGroups {
			if previousNg.Name == updatingNg.Name {
				ngNewCopy := *updatingNg
				ngOldCopy := *previousNg
				commonNewNodeGroups = append(commonNewNodeGroups, &ngNewCopy)
				commonOldNodeGroups = append(commonOldNodeGroups, &ngOldCopy)
				break
			}
		}
	}
	return commonNewNodeGroups, commonOldNodeGroups
}

func GetNodeGroupNames(nodeGroups []*NodeGroup) []string {
	ngNames := []string{}
	for _, ng := range nodeGroups {
		ngNames = append(ngNames, ng.Name)
	}
	return ngNames
}

func GetNodeGroupAvailabilities(nodeGroups []*NodeGroup) []bool {
	ngAvailability := []bool{}
	for _, ng := range nodeGroups {
		ngAvailability = append(ngAvailability, ng.Spot)
	}
	return ngAvailability
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

func validateCIDR(cidr string) (string, error) {
	_, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", errors.WithStack(err)
	}

	return cidr, nil
}

func validateInstanceType(instanceType string) (string, error) {
	if err := aws.CheckValidInstanceType(instanceType); err != nil {
		return "", err
	}

	parsedType, err := aws.ParseInstanceType(instanceType)
	if err != nil {
		return "", err
	}

	if parsedType.Size == "nano" || parsedType.Size == "micro" {
		return "", ErrorInstanceTypeTooSmall(instanceType)
	}

	isSupportedByNLB, err := aws.IsInstanceSupportedByNLB(instanceType)
	if err != nil {
		return "", err
	}
	if !isSupportedByNLB {
		return "", ErrorInstanceTypeNotSupportedByCortex(instanceType)
	}

	isARM, err := aws.IsARMInstance(instanceType)
	if err != nil {
		return "", err
	}
	if isARM {
		return "", ErrorARMInstancesNotSupported(instanceType)
	}

	isAMDGPU, err := aws.IsAMDGPUInstance(instanceType)
	if err != nil {
		return "", err
	}
	if isAMDGPU {
		return "", ErrorAMDGPUInstancesNotSupported(instanceType)
	}

	if _, ok := awsutils.InstanceNetworkingLimits[instanceType]; !ok {
		return "", ErrorInstanceTypeNotSupportedByCortex(instanceType)
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

func (ng *NodeGroup) DeepCopy() (NodeGroup, error) {
	bytes, err := yaml.Marshal(ng)
	if err != nil {
		return NodeGroup{}, err
	}

	deepCopied := NodeGroup{}
	err = yaml.Unmarshal(bytes, &deepCopied)
	if err != nil {
		return NodeGroup{}, err
	}

	return deepCopied, nil
}

func (ng *NodeGroup) Hash() (string, error) {
	bytes, err := yaml.Marshal(ng)
	if err != nil {
		return "", err
	}

	hash := sha256.New()
	hash.Write(bytes)
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func (ng *NodeGroup) MaxPossibleOnDemandInstances() int64 {
	if !ng.Spot || ng.SpotConfig == nil {
		return ng.MaxInstances
	}

	onDemandBaseCap, onDemandPctAboveBaseCap := ng.SpotConfigOnDemandValues()
	return onDemandBaseCap + int64(math.Ceil(float64(onDemandPctAboveBaseCap)/100*float64(ng.MaxInstances-onDemandBaseCap)))
}

func (ng *NodeGroup) MaxPossibleSpotInstances() int64 {
	if !ng.Spot {
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
	event := make(map[string]interface{})

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
	if !strings.HasPrefix(cc.ImageControllerManager, "cortexlabs/") {
		event["image_operator_controller_manager._is_custom"] = true
	}
	if !strings.HasPrefix(cc.ImageManager, "cortexlabs/") {
		event["image_manager._is_custom"] = true
	}
	if !strings.HasPrefix(cc.ImageKubexit, "cortexlabs/") {
		event["image_kubexit._is_custom"] = true
	}
	if !strings.HasPrefix(cc.ImageProxy, "cortexlabs/") {
		event["image_proxy._is_custom"] = true
	}
	if !strings.HasPrefix(cc.ImageAsyncGateway, "cortexlabs/") {
		event["image_async_gateway._is_custom"] = true
	}
	if !strings.HasPrefix(cc.ImageEnqueuer, "cortexlabs/") {
		event["image_enqueuer._is_custom"] = true
	}
	if !strings.HasPrefix(cc.ImageDequeuer, "cortexlabs/") {
		event["image_dequeuer._is_custom"] = true
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
		event[nodeGroupKey("priority")] = ng.Priority
		event[nodeGroupKey("instance_volume_size")] = ng.InstanceVolumeSize
		event[nodeGroupKey("instance_volume_type")] = ng.InstanceVolumeType
		if ng.InstanceVolumeIOPS != nil {
			event[nodeGroupKey("instance_volume_iops.is_defined")] = true
			event[nodeGroupKey("instance_volume_iops")] = *ng.InstanceVolumeIOPS
		}
		if ng.InstanceVolumeThroughput != nil {
			event[nodeGroupKey("instance_volume_throughput.is_defined")] = true
			event[nodeGroupKey("instance_volume_throughput")] = *ng.InstanceVolumeThroughput
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

func (mc *ManagedConfig) GetNodeGroupByName(name string) *NodeGroup {
	for _, ng := range mc.NodeGroups {
		if ng.Name == name {
			matchedNodeGroup := *ng
			return &matchedNodeGroup
		}
	}

	return nil
}

func (mc *ManagedConfig) GetNodeGroupNames() []string {
	allNodeGroupNames := []string{}
	for _, ng := range mc.NodeGroups {
		allNodeGroupNames = append(allNodeGroupNames, ng.Name)
	}

	return allNodeGroupNames
}

func BucketName(accountID, clusterName, region string) string {
	bucketID := hash.String(accountID + region)[:8] // this is to "guarantee" a globally unique name
	return clusterName + "-" + bucketID
}

func validateClusterName(clusterName string) (string, error) {
	if !_strictS3BucketRegex.MatchString(clusterName) {
		return "", errors.Wrap(ErrorDidNotMatchStrictS3Regex(), clusterName)
	}
	return clusterName, nil
}

func validateImageVersion(image string) (string, error) {
	return cr.ValidateImageVersion(image, consts.CortexVersion)
}
