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
	"fmt"
	"regexp"
	"strings"

	"github.com/aws/amazon-vpc-cni-k8s/pkg/awsutils"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/apigatewayv2"
	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/hash"
	libmath "github.com/cortexlabs/cortex/pkg/lib/math"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/prompt"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/table"
)

const ClusterNameTag = "cortex.dev/cluster-name"

var (
	_spotInstanceDistributionLength = 2
	_maxInstancePools               = 20
	// This regex is stricter than the actual S3 rules
	_strictS3BucketRegex = regexp.MustCompile(`^([a-z0-9])+(-[a-z0-9]+)*$`)
)

type Config struct {
	InstanceType               *string            `json:"instance_type" yaml:"instance_type"`
	MinInstances               *int64             `json:"min_instances" yaml:"min_instances"`
	MaxInstances               *int64             `json:"max_instances" yaml:"max_instances"`
	InstanceVolumeSize         int64              `json:"instance_volume_size" yaml:"instance_volume_size"`
	InstanceVolumeType         VolumeType         `json:"instance_volume_type" yaml:"instance_volume_type"`
	InstanceVolumeIOPS         *int64             `json:"instance_volume_iops" yaml:"instance_volume_iops"`
	Tags                       map[string]string  `json:"tags" yaml:"tags"`
	Spot                       *bool              `json:"spot" yaml:"spot"`
	SpotConfig                 *SpotConfig        `json:"spot_config" yaml:"spot_config"`
	ClusterName                string             `json:"cluster_name" yaml:"cluster_name"`
	Region                     *string            `json:"region" yaml:"region"`
	AvailabilityZones          []string           `json:"availability_zones" yaml:"availability_zones"`
	SSLCertificateARN          *string            `json:"ssl_certificate_arn,omitempty" yaml:"ssl_certificate_arn,omitempty"`
	Bucket                     string             `json:"bucket" yaml:"bucket"`
	LogGroup                   string             `json:"log_group" yaml:"log_group"`
	SubnetVisibility           SubnetVisibility   `json:"subnet_visibility" yaml:"subnet_visibility"`
	NATGateway                 NATGateway         `json:"nat_gateway" yaml:"nat_gateway"`
	APILoadBalancerScheme      LoadBalancerScheme `json:"api_load_balancer_scheme" yaml:"api_load_balancer_scheme"`
	OperatorLoadBalancerScheme LoadBalancerScheme `json:"operator_load_balancer_scheme" yaml:"operator_load_balancer_scheme"`
	APIGatewaySetting          APIGatewaySetting  `json:"api_gateway" yaml:"api_gateway"`
	Telemetry                  bool               `json:"telemetry" yaml:"telemetry"`
	ImageOperator              string             `json:"image_operator" yaml:"image_operator"`
	ImageManager               string             `json:"image_manager" yaml:"image_manager"`
	ImageDownloader            string             `json:"image_downloader" yaml:"image_downloader"`
	ImageRequestMonitor        string             `json:"image_request_monitor" yaml:"image_request_monitor"`
	ImageClusterAutoscaler     string             `json:"image_cluster_autoscaler" yaml:"image_cluster_autoscaler"`
	ImageMetricsServer         string             `json:"image_metrics_server" yaml:"image_metrics_server"`
	ImageInferentia            string             `json:"image_inferentia" yaml:"image_inferentia"`
	ImageNeuronRTD             string             `json:"image_neuron_rtd" yaml:"image_neuron_rtd"`
	ImageNvidia                string             `json:"image_nvidia" yaml:"image_nvidia"`
	ImageFluentd               string             `json:"image_fluentd" yaml:"image_fluentd"`
	ImageStatsd                string             `json:"image_statsd" yaml:"image_statsd"`
	ImageIstioProxy            string             `json:"image_istio_proxy" yaml:"image_istio_proxy"`
	ImageIstioPilot            string             `json:"image_istio_pilot" yaml:"image_istio_pilot"`
	ImageIstioCitadel          string             `json:"image_istio_citadel" yaml:"image_istio_citadel"`
	ImageIstioGalley           string             `json:"image_istio_galley" yaml:"image_istio_galley"`
}

type SpotConfig struct {
	InstanceDistribution                []string `json:"instance_distribution" yaml:"instance_distribution"`
	OnDemandBaseCapacity                *int64   `json:"on_demand_base_capacity" yaml:"on_demand_base_capacity"`
	OnDemandPercentageAboveBaseCapacity *int64   `json:"on_demand_percentage_above_base_capacity" yaml:"on_demand_percentage_above_base_capacity"`
	MaxPrice                            *float64 `json:"max_price" yaml:"max_price"`
	InstancePools                       *int64   `json:"instance_pools" yaml:"instance_pools"`
	OnDemandBackup                      *bool    `json:"on_demand_backup" yaml:"on_demand_backup"`
}

type InternalConfig struct {
	Config

	// Populated by operator
	ID                 string                    `json:"id"`
	APIVersion         string                    `json:"api_version"`
	OperatorInCluster  bool                      `json:"operator_in_cluster"`
	InstanceMetadata   aws.InstanceMetadata      `json:"instance_metadata"`
	APIGateway         *apigatewayv2.Api         `json:"api_gateway"`
	VPCLink            *apigatewayv2.VpcLink     `json:"vpc_link"`
	VPCLinkIntegration *apigatewayv2.Integration `json:"vpc_link_integration"`
}

// The bare minimum to identify a cluster
type AccessConfig struct {
	ClusterName  *string `json:"cluster_name" yaml:"cluster_name"`
	Region       *string `json:"region" yaml:"region"`
	ImageManager string  `json:"image_manager" yaml:"image_manager"`
}

var UserValidation = &cr.StructValidation{
	Required: true,
	StructFieldValidations: []*cr.StructFieldValidation{
		{
			StructField: "InstanceType",
			StringPtrValidation: &cr.StringPtrValidation{
				Validator: validateInstanceType,
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
			StructField: "Tags",
			StringMapValidation: &cr.StringMapValidation{
				AllowExplicitNull:  true,
				AllowEmpty:         true,
				ConvertNullToEmpty: true,
			},
		},
		{
			StructField: "SSLCertificateARN",
			StringPtrValidation: &cr.StringPtrValidation{
				AllowExplicitNull: true,
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
			BoolPtrValidation: &cr.BoolPtrValidation{
				Default: pointer.Bool(false),
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
					{
						StructField: "OnDemandBackup",
						BoolPtrValidation: &cr.BoolPtrValidation{
							Default: pointer.Bool(true),
						},
					},
				},
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
			StructField: "Region",
			StringPtrValidation: &cr.StringPtrValidation{
				Validator: RegionValidator,
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
			StructField: "Bucket",
			StringValidation: &cr.StringValidation{
				AllowEmpty:       true,
				TreatNullAsEmpty: true,
				Validator:        validateBucketNameOrEmpty,
			},
		},
		{
			StructField: "LogGroup",
			StringValidation: &cr.StringValidation{
				MaxLength: 63,
			},
			DefaultField: "ClusterName",
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
			StructField: "NATGateway",
			StringValidation: &cr.StringValidation{
				AllowedValues: NATGatewayStrings(),
			},
			Parser: func(str string) (interface{}, error) {
				return NATGatewayFromString(str), nil
			},
			DefaultField: "SubnetVisibility",
			DefaultFieldFunc: func(val interface{}) interface{} {
				if val.(SubnetVisibility) == PublicSubnetVisibility {
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
			StructField: "APIGatewaySetting",
			StringValidation: &cr.StringValidation{
				AllowedValues: APIGatewaySettingStrings(),
				Default:       EnabledAPIGatewaySetting.String(),
			},
			Parser: func(str string) (interface{}, error) {
				return APIGatewaySettingFromString(str), nil
			},
		},
		{
			StructField: "ImageOperator",
			StringValidation: &cr.StringValidation{
				Default:   "cortexlabs/operator:" + consts.CortexVersion,
				Validator: validateImageVersion,
			},
		},
		{
			StructField: "ImageManager",
			StringValidation: &cr.StringValidation{
				Default:   "cortexlabs/manager:" + consts.CortexVersion,
				Validator: validateImageVersion,
			},
		},
		{
			StructField: "ImageDownloader",
			StringValidation: &cr.StringValidation{
				Default:   "cortexlabs/downloader:" + consts.CortexVersion,
				Validator: validateImageVersion,
			},
		},
		{
			StructField: "ImageRequestMonitor",
			StringValidation: &cr.StringValidation{
				Default:   "cortexlabs/request-monitor:" + consts.CortexVersion,
				Validator: validateImageVersion,
			},
		},
		{
			StructField: "ImageClusterAutoscaler",
			StringValidation: &cr.StringValidation{
				Default:   "cortexlabs/cluster-autoscaler:" + consts.CortexVersion,
				Validator: validateImageVersion,
			},
		},
		{
			StructField: "ImageMetricsServer",
			StringValidation: &cr.StringValidation{
				Default:   "cortexlabs/metrics-server:" + consts.CortexVersion,
				Validator: validateImageVersion,
			},
		},
		{
			StructField: "ImageInferentia",
			StringValidation: &cr.StringValidation{
				Default:   "cortexlabs/inferentia:" + consts.CortexVersion,
				Validator: validateImageVersion,
			},
		},
		{
			StructField: "ImageNeuronRTD",
			StringValidation: &cr.StringValidation{
				Default:   "cortexlabs/neuron-rtd:" + consts.CortexVersion,
				Validator: validateImageVersion,
			},
		},
		{
			StructField: "ImageNvidia",
			StringValidation: &cr.StringValidation{
				Default:   "cortexlabs/nvidia:" + consts.CortexVersion,
				Validator: validateImageVersion,
			},
		},
		{
			StructField: "ImageFluentd",
			StringValidation: &cr.StringValidation{
				Default:   "cortexlabs/fluentd:" + consts.CortexVersion,
				Validator: validateImageVersion,
			},
		},
		{
			StructField: "ImageStatsd",
			StringValidation: &cr.StringValidation{
				Default:   "cortexlabs/statsd:" + consts.CortexVersion,
				Validator: validateImageVersion,
			},
		},
		{
			StructField: "ImageIstioProxy",
			StringValidation: &cr.StringValidation{
				Default:   "cortexlabs/istio-proxy:" + consts.CortexVersion,
				Validator: validateImageVersion,
			},
		},
		{
			StructField: "ImageIstioPilot",
			StringValidation: &cr.StringValidation{
				Default:   "cortexlabs/istio-pilot:" + consts.CortexVersion,
				Validator: validateImageVersion,
			},
		},
		{
			StructField: "ImageIstioCitadel",
			StringValidation: &cr.StringValidation{
				Default:   "cortexlabs/istio-citadel:" + consts.CortexVersion,
				Validator: validateImageVersion,
			},
		},
		{
			StructField: "ImageIstioGalley",
			StringValidation: &cr.StringValidation{
				Default:   "cortexlabs/istio-galley:" + consts.CortexVersion,
				Validator: validateImageVersion,
			},
		},
		// Extra keys that exist in the cluster config file
		{
			Key: "aws_access_key_id",
			Nil: true,
		},
		{
			Key: "aws_secret_access_key",
			Nil: true,
		},
		{
			Key: "cortex_aws_access_key_id",
			Nil: true,
		},
		{
			Key: "cortex_aws_secret_access_key",
			Nil: true,
		},
	},
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

func validateImageVersion(image string) (string, error) {
	return cr.ValidateImageVersion(image, consts.CortexVersion)
}

var Validation = &cr.StructValidation{
	StructFieldValidations: append(UserValidation.StructFieldValidations,
		&cr.StructFieldValidation{
			StructField: "Telemetry",
			BoolValidation: &cr.BoolValidation{
				Default: true,
			},
		},
	),
}

var AccessValidation = &cr.StructValidation{
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
			StructField: "Region",
			StringPtrValidation: &cr.StringPtrValidation{
				Validator: RegionValidator,
			},
		},
		{
			StructField: "ImageManager",
			StringValidation: &cr.StringValidation{
				Default:   "cortexlabs/manager:" + consts.CortexVersion,
				Validator: validateImageVersion,
			},
		},
	},
}

func (cc *Config) ToAccessConfig() AccessConfig {
	clusterName := cc.ClusterName
	region := *cc.Region
	return AccessConfig{
		ClusterName:  &clusterName,
		Region:       &region,
		ImageManager: cc.ImageManager,
	}
}

func SQSNamePrefix(clusterName string) string {
	// 10 was chosen to make sure that other identifiers can be added to the full queue name before reaching the 80 char SQS name limit
	return hash.String(clusterName)[:10] + "-"
}

// returns hash of cluster name and adds trailing "-"
func (cc *Config) SQSNamePrefix() string {
	return SQSNamePrefix(cc.ClusterName)
}

func (cc *Config) Validate(awsClient *aws.Client) error {
	fmt.Print("verifying your configuration ...\n\n")

	if *cc.MinInstances > *cc.MaxInstances {
		return ErrorMinInstancesGreaterThanMax(*cc.MinInstances, *cc.MaxInstances)
	}

	if cc.SubnetVisibility == PrivateSubnetVisibility && cc.NATGateway == NoneNATGateway {
		return ErrorNATRequiredWithPrivateSubnetVisibility()
	}

	if cc.Bucket == "" {
		accountID, _, err := awsClient.GetCachedAccountID()
		if err != nil {
			return err
		}

		bucketID := hash.String(accountID + *cc.Region)[:10]

		defaultBucket := cc.ClusterName + "-" + bucketID
		if len(defaultBucket) > 63 {
			defaultBucket = defaultBucket[:63]
		}
		if strings.HasSuffix(defaultBucket, "-") {
			defaultBucket = defaultBucket[:len(defaultBucket)-1]
		}

		cc.Bucket = defaultBucket
	} else {
		bucketRegion, _ := aws.GetBucketRegion(cc.Bucket)
		if bucketRegion != "" && bucketRegion != *cc.Region { // if the bucket didn't exist, we will create it in the correct region, so there is no error
			return ErrorS3RegionDiffersFromCluster(cc.Bucket, bucketRegion, *cc.Region)
		}
	}

	primaryInstanceType := *cc.InstanceType
	if _, ok := aws.InstanceMetadatas[*cc.Region][primaryInstanceType]; !ok {
		return errors.Wrap(ErrorInstanceTypeNotSupportedInRegion(primaryInstanceType, *cc.Region), InstanceTypeKey)
	}

	if cc.SSLCertificateARN != nil {
		exists, err := awsClient.DoesCertificateExist(*cc.SSLCertificateARN)
		if err != nil {
			return errors.Wrap(err, SSLCertificateARNKey)
		}

		if !exists {
			return errors.Wrap(ErrorSSLCertificateARNNotFound(*cc.SSLCertificateARN, *cc.Region), SSLCertificateARNKey)
		}
	}

	// Throw error if IOPS defined for other storage than io1
	if cc.InstanceVolumeType != IO1VolumeType && cc.InstanceVolumeIOPS != nil {
		return ErrorIOPSNotSupported(cc.InstanceVolumeType)
	}

	if cc.InstanceVolumeType == IO1VolumeType && cc.InstanceVolumeIOPS != nil {
		if *cc.InstanceVolumeIOPS > cc.InstanceVolumeSize*50 {
			return ErrorIOPSTooLarge(*cc.InstanceVolumeIOPS, cc.InstanceVolumeSize)
		}
	}

	if aws.EBSMetadatas[*cc.Region][cc.InstanceVolumeType.String()].IOPSConfigurable && cc.InstanceVolumeIOPS == nil {
		cc.InstanceVolumeIOPS = pointer.Int64(libmath.MinInt64(cc.InstanceVolumeSize*50, 3000))
	}

	if err := awsClient.VerifyInstanceQuota(primaryInstanceType); err != nil {
		// Skip AWS errors, since some regions (e.g. eu-north-1) do not support this API
		if _, ok := errors.CauseOrSelf(err).(awserr.Error); !ok {
			return errors.Wrap(err, InstanceTypeKey)
		}
	}

	if cc.Tags[ClusterNameTag] != "" && cc.Tags[ClusterNameTag] != cc.ClusterName {
		return ErrorCantOverrideDefaultTag()
	}
	cc.Tags[ClusterNameTag] = cc.ClusterName

	if err := cc.validateAvailabilityZones(awsClient); err != nil {
		return errors.Wrap(err, AvailabilityZonesKey)
	}

	if cc.Spot != nil && *cc.Spot {
		cc.FillEmptySpotFields(awsClient)

		primaryInstance := aws.InstanceMetadatas[*cc.Region][primaryInstanceType]

		for _, instanceType := range cc.SpotConfig.InstanceDistribution {
			if instanceType == primaryInstanceType {
				continue
			}
			if _, ok := aws.InstanceMetadatas[*cc.Region][instanceType]; !ok {
				return errors.Wrap(ErrorInstanceTypeNotSupportedInRegion(instanceType, *cc.Region), SpotConfigKey, InstanceDistributionKey)
			}

			instanceMetadata := aws.InstanceMetadatas[*cc.Region][instanceType]
			err := CheckSpotInstanceCompatibility(primaryInstance, instanceMetadata)
			if err != nil {
				return errors.Wrap(err, SpotConfigKey, InstanceDistributionKey)
			}

			spotInstancePrice, awsErr := awsClient.SpotInstancePrice(instanceMetadata.Region, instanceMetadata.Type)
			if awsErr == nil {
				if err := CheckSpotInstancePriceCompatibility(primaryInstance, instanceMetadata, cc.SpotConfig.MaxPrice, spotInstancePrice); err != nil {
					return errors.Wrap(err, SpotConfigKey, InstanceDistributionKey)
				}
			}
		}

		if cc.SpotConfig.OnDemandBaseCapacity != nil && *cc.SpotConfig.OnDemandBaseCapacity > *cc.MaxInstances {
			return ErrorOnDemandBaseCapacityGreaterThanMax(*cc.SpotConfig.OnDemandBaseCapacity, *cc.MaxInstances)
		}
	} else {
		if cc.SpotConfig != nil {
			return ErrorConfiguredWhenSpotIsNotEnabled(SpotConfigKey)
		}
	}

	return nil
}

func CheckCortexSupport(instanceMetadata aws.InstanceMetadata) error {
	if strings.HasSuffix(instanceMetadata.Type, "nano") ||
		strings.HasSuffix(instanceMetadata.Type, "micro") {
		return ErrorInstanceTypeTooSmall()
	}

	if _, ok := awsutils.InstanceENIsAvailable[instanceMetadata.Type]; !ok {
		return ErrorInstanceTypeNotSupported(instanceMetadata.Type)
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

func AutoGenerateSpotConfig(awsClient *aws.Client, spotConfig *SpotConfig, region string, instanceType string) error {
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

	if spotConfig.OnDemandBackup == nil {
		spotConfig.OnDemandBackup = pointer.Bool(true)
	}

	if spotConfig.InstancePools == nil {
		if len(spotConfig.InstanceDistribution) < _maxInstancePools {
			spotConfig.InstancePools = pointer.Int64(int64(len(spotConfig.InstanceDistribution)))
		} else {
			spotConfig.InstancePools = pointer.Int64(int64(_maxInstancePools))
		}
	}

	return nil
}

func (cc *Config) FillEmptySpotFields(awsClient *aws.Client) error {
	if cc.SpotConfig == nil {
		cc.SpotConfig = &SpotConfig{}
	}
	err := AutoGenerateSpotConfig(awsClient, cc.SpotConfig, *cc.Region, *cc.InstanceType)
	if err != nil {
		return err
	}
	return nil
}

func applyPromptDefaults(defaults Config) *Config {
	defaultConfig := &Config{
		Region:       pointer.String("us-east-1"),
		InstanceType: pointer.String("m5.large"),
		MinInstances: pointer.Int64(1),
		MaxInstances: pointer.Int64(5),
		Spot:         pointer.Bool(true),
	}

	if defaults.Region != nil {
		defaultConfig.Region = defaults.Region
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
	if defaults.Spot != nil {
		defaultConfig.Spot = defaults.Spot
	}

	return defaultConfig
}

func RegionPrompt(clusterConfig *Config, disallowPrompt bool) error {
	defaults := applyPromptDefaults(*clusterConfig)

	if disallowPrompt {
		if clusterConfig.Region == nil {
			clusterConfig.Region = defaults.Region
		}
		return nil
	}

	regionPrompt := &cr.PromptValidation{
		SkipNonNilFields: true,
		PromptItemValidations: []*cr.PromptItemValidation{
			{
				StructField: "Region",
				PromptOpts: &prompt.Options{
					Prompt: RegionUserKey,
				},
				StringPtrValidation: &cr.StringPtrValidation{
					Validator: RegionValidator,
					Default:   defaults.Region,
				},
			},
		},
	}
	err := cr.ReadPrompt(clusterConfig, regionPrompt)
	if err != nil {
		return err
	}

	return nil
}

func InstallPrompt(clusterConfig *Config, disallowPrompt bool) error {
	defaults := applyPromptDefaults(*clusterConfig)

	if disallowPrompt {
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
					Prompt: "aws instance type",
				},
				StringPtrValidation: &cr.StringPtrValidation{
					Required:  true,
					Default:   defaults.InstanceType,
					Validator: validateInstanceType,
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

func ConfigurePrompt(userClusterConfig *Config, cachedClusterConfig *Config, skipPopulatedFields bool, disallowPrompt bool) error {
	defaults := applyPromptDefaults(*cachedClusterConfig)

	if disallowPrompt {
		if userClusterConfig.MinInstances == nil {
			if cachedClusterConfig.MinInstances != nil {
				userClusterConfig.MinInstances = cachedClusterConfig.MinInstances
			} else {
				userClusterConfig.MinInstances = defaults.MinInstances
			}
		}
		if userClusterConfig.MaxInstances == nil {
			if cachedClusterConfig.MaxInstances != nil {
				userClusterConfig.MaxInstances = cachedClusterConfig.MaxInstances
			} else {
				userClusterConfig.MaxInstances = defaults.MaxInstances
			}
		}
		return nil
	}

	remainingPrompts := &cr.PromptValidation{
		SkipNonNilFields: skipPopulatedFields,
		PromptItemValidations: []*cr.PromptItemValidation{
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

	err := cr.ReadPrompt(userClusterConfig, remainingPrompts)
	if err != nil {
		return err
	}

	return nil
}

var AccessPromptValidation = &cr.PromptValidation{
	SkipNonNilFields: true,
	PromptItemValidations: []*cr.PromptItemValidation{
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
		{
			StructField: "Region",
			PromptOpts: &prompt.Options{
				Prompt: RegionUserKey,
			},
			StringPtrValidation: &cr.StringPtrValidation{
				Validator: RegionValidator,
				Default:   pointer.String("us-east-1"),
			},
		},
	},
}

func validateClusterName(clusterName string) (string, error) {
	if !_strictS3BucketRegex.MatchString(clusterName) {
		return "", errors.Wrap(ErrorDidNotMatchStrictS3Regex(), clusterName)
	}
	return clusterName, nil
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

	err := CheckCortexSupport(*foundInstance)
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
	errs := cr.Struct(cc, emptyMap, Validation)
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

func DefaultAccessConfig() (*AccessConfig, error) {
	accessConfig := &AccessConfig{}
	var emptyMap interface{} = map[interface{}]interface{}{}
	errs := cr.Struct(accessConfig, emptyMap, AccessValidation)
	if errors.HasError(errs) {
		return nil, errors.FirstError(errs...)
	}
	return accessConfig, nil
}

func (cc *InternalConfig) UserTable() table.KeyValuePairs {
	var items table.KeyValuePairs

	items.Add(APIVersionUserKey, cc.APIVersion)
	items.AddAll(cc.Config.UserTable())
	return items
}

func (cc *InternalConfig) UserStr() string {
	return cc.UserTable().String()
}

func (cc *Config) UserTable() table.KeyValuePairs {
	var items table.KeyValuePairs

	items.Add(ClusterNameUserKey, cc.ClusterName)
	items.Add(RegionUserKey, *cc.Region)
	if len(cc.AvailabilityZones) > 0 {
		items.Add(AvailabilityZonesUserKey, cc.AvailabilityZones)
	}
	items.Add(BucketUserKey, cc.Bucket)
	items.Add(InstanceTypeUserKey, *cc.InstanceType)
	items.Add(MinInstancesUserKey, *cc.MinInstances)
	items.Add(MaxInstancesUserKey, *cc.MaxInstances)
	items.Add(TagsUserKey, s.ObjFlat(cc.Tags))
	if cc.SSLCertificateARN != nil {
		items.Add(SSLCertificateARNUserKey, *cc.SSLCertificateARN)
	}
	items.Add(InstanceVolumeSizeUserKey, cc.InstanceVolumeSize)
	items.Add(InstanceVolumeTypeUserKey, cc.InstanceVolumeType)
	items.Add(InstanceVolumeIOPSUserKey, cc.InstanceVolumeIOPS)
	items.Add(SpotUserKey, s.YesNo(*cc.Spot))

	if cc.Spot != nil && *cc.Spot {
		items.Add(InstanceDistributionUserKey, cc.SpotConfig.InstanceDistribution)
		items.Add(OnDemandBaseCapacityUserKey, *cc.SpotConfig.OnDemandBaseCapacity)
		items.Add(OnDemandPercentageAboveBaseCapacityUserKey, *cc.SpotConfig.OnDemandPercentageAboveBaseCapacity)
		items.Add(MaxPriceUserKey, *cc.SpotConfig.MaxPrice)
		items.Add(InstancePoolsUserKey, *cc.SpotConfig.InstancePools)
		items.Add(OnDemandBackupUserKey, s.YesNo(*cc.SpotConfig.OnDemandBackup))
	}
	items.Add(LogGroupUserKey, cc.LogGroup)
	items.Add(SubnetVisibilityUserKey, cc.SubnetVisibility)
	items.Add(NATGatewayUserKey, cc.NATGateway)
	items.Add(APILoadBalancerSchemeUserKey, cc.APILoadBalancerScheme)
	items.Add(OperatorLoadBalancerSchemeUserKey, cc.OperatorLoadBalancerScheme)
	items.Add(APIGatewaySettingUserKey, cc.APIGatewaySetting)
	items.Add(TelemetryUserKey, cc.Telemetry)
	items.Add(ImageOperatorUserKey, cc.ImageOperator)
	items.Add(ImageManagerUserKey, cc.ImageManager)
	items.Add(ImageDownloaderUserKey, cc.ImageDownloader)
	items.Add(ImageRequestMonitorUserKey, cc.ImageRequestMonitor)
	items.Add(ImageClusterAutoscalerUserKey, cc.ImageClusterAutoscaler)
	items.Add(ImageMetricsServerUserKey, cc.ImageMetricsServer)
	items.Add(ImageInferentiaUserKey, cc.ImageInferentia)
	items.Add(ImageNeuronRTDUserKey, cc.ImageNeuronRTD)
	items.Add(ImageNvidiaUserKey, cc.ImageNvidia)
	items.Add(ImageFluentdUserKey, cc.ImageFluentd)
	items.Add(ImageStatsdUserKey, cc.ImageStatsd)
	items.Add(ImageIstioProxyUserKey, cc.ImageIstioProxy)
	items.Add(ImageIstioPilotUserKey, cc.ImageIstioPilot)
	items.Add(ImageIstioCitadelUserKey, cc.ImageIstioCitadel)
	items.Add(ImageIstioGalleyUserKey, cc.ImageIstioGalley)

	return items
}

func (cc *Config) UserStr() string {
	return cc.UserTable().String()
}
