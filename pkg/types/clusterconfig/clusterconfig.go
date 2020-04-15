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
	"sort"
	"strings"

	"github.com/aws/amazon-vpc-cni-k8s/pkg/awsutils"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/hash"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/prompt"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/table"
)

var (
	_spotInstanceDistributionLength = 2
	_maxInstancePools               = 20

	// This regex is stricter than the actual S3 rules
	_strictS3BucketRegex = regexp.MustCompile(`^([a-z0-9])+(-[a-z0-9]+)*$`)
)

type Config struct {
	InstanceType           *string     `json:"instance_type" yaml:"instance_type"`
	MinInstances           *int64      `json:"min_instances" yaml:"min_instances"`
	MaxInstances           *int64      `json:"max_instances" yaml:"max_instances"`
	InstanceVolumeSize     int64       `json:"instance_volume_size" yaml:"instance_volume_size"`
	Spot                   *bool       `json:"spot" yaml:"spot"`
	SpotConfig             *SpotConfig `json:"spot_config" yaml:"spot_config"`
	ClusterName            string      `json:"cluster_name" yaml:"cluster_name"`
	Region                 *string     `json:"region" yaml:"region"`
	AvailabilityZones      []string    `json:"availability_zones" yaml:"availability_zones"`
	Bucket                 string      `json:"bucket" yaml:"bucket"`
	LogGroup               string      `json:"log_group" yaml:"log_group"`
	Telemetry              bool        `json:"telemetry" yaml:"telemetry"`
	ImageOperator          string      `json:"image_operator" yaml:"image_operator"`
	ImageManager           string      `json:"image_manager" yaml:"image_manager"`
	ImageDownloader        string      `json:"image_downloader" yaml:"image_downloader"`
	ImageRequestMonitor    string      `json:"image_request_monitor" yaml:"image_request_monitor"`
	ImageClusterAutoscaler string      `json:"image_cluster_autoscaler" yaml:"image_cluster_autoscaler"`
	ImageMetricsServer     string      `json:"image_metrics_server" yaml:"image_metrics_server"`
	ImageNvidia            string      `json:"image_nvidia" yaml:"image_nvidia"`
	ImageFluentd           string      `json:"image_fluentd" yaml:"image_fluentd"`
	ImageStatsd            string      `json:"image_statsd" yaml:"image_statsd"`
	ImageIstioProxy        string      `json:"image_istio_proxy" yaml:"image_istio_proxy"`
	ImageIstioPilot        string      `json:"image_istio_pilot" yaml:"image_istio_pilot"`
	ImageIstioCitadel      string      `json:"image_istio_citadel" yaml:"image_istio_citadel"`
	ImageIstioGalley       string      `json:"image_istio_galley" yaml:"image_istio_galley"`
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
	ID                string               `json:"id"`
	APIVersion        string               `json:"api_version"`
	OperatorInCluster bool                 `json:"operator_in_cluster"`
	InstanceMetadata  aws.InstanceMetadata `json:"instance_metadata"`
}

// The bare minimum to identify a cluster
type AccessConfig struct {
	ClusterName  *string `json:"cluster_name" yaml:"cluster_name"`
	Region       *string `json:"region" yaml:"region"`
	ImageManager string  `json:"image_manager" yaml:"image_manager"`
}

var UserValidation = &cr.StructValidation{
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
				Validator: validateRegion,
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

func validateRegion(region string) (string, error) {
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
				Validator: validateRegion,
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

func (cc *Config) Validate(awsClient *aws.Client) error {
	fmt.Print("verifying your configuration...\n\n")

	if *cc.MinInstances > *cc.MaxInstances {
		return ErrorMinInstancesGreaterThanMax(*cc.MinInstances, *cc.MaxInstances)
	}

	bucketRegion, _ := aws.GetBucketRegion(cc.Bucket)
	if bucketRegion != "" && bucketRegion != *cc.Region { // if the bucket didn't exist, we will create it in the correct region, so there is no error
		return ErrorS3RegionDiffersFromCluster(cc.Bucket, bucketRegion, *cc.Region)
	}

	if _, ok := aws.InstanceMetadatas[*cc.Region][*cc.InstanceType]; !ok {
		return errors.Wrap(ErrorInstanceTypeNotSupportedInRegion(*cc.InstanceType, *cc.Region), InstanceTypeKey)
	}

	if err := awsClient.VerifyInstanceQuota(*cc.InstanceType); err != nil {
		// Skip AWS errors, since some regions (e.g. eu-north-1) do not support this API
		if _, ok := errors.CauseOrSelf(err).(awserr.Error); !ok {
			return errors.Wrap(err, InstanceTypeKey)
		}
	}

	if err := cc.validateAvailabilityZones(awsClient); err != nil {
		return errors.Wrap(err, AvailabilityZonesKey)
	}

	if cc.Spot != nil && *cc.Spot {
		cc.AutoFillSpot(awsClient)
		chosenInstance := aws.InstanceMetadatas[*cc.Region][*cc.InstanceType]
		compatibleSpots := CompatibleSpotInstances(awsClient, chosenInstance, cc.SpotConfig.MaxPrice, _spotInstanceDistributionLength)
		if len(compatibleSpots) == 0 {
			return errors.Wrap(ErrorNoCompatibleSpotInstanceFound(chosenInstance.Type), InstanceTypeKey)
		}

		compatibleInstanceCount := 0
		for _, instanceType := range cc.SpotConfig.InstanceDistribution {
			if instanceType == *cc.InstanceType {
				continue
			}
			if _, ok := aws.InstanceMetadatas[*cc.Region][instanceType]; !ok {
				return errors.Wrap(ErrorInstanceTypeNotSupportedInRegion(instanceType, *cc.Region), InstanceDistributionKey)
			}

			instanceMetadata := aws.InstanceMetadatas[*cc.Region][instanceType]
			err := CheckSpotInstanceCompatibility(chosenInstance, instanceMetadata)
			if err != nil {
				return errors.Wrap(err, InstanceDistributionKey)
			}

			spotInstancePrice, awsErr := awsClient.SpotInstancePrice(instanceMetadata.Region, instanceMetadata.Type)
			if awsErr == nil {
				if err := CheckSpotInstancePriceCompatibility(chosenInstance, instanceMetadata, cc.SpotConfig.MaxPrice, spotInstancePrice); err != nil {
					return errors.Wrap(err, InstanceDistributionKey)
				}
			}

			compatibleInstanceCount++
		}

		if compatibleInstanceCount == 0 {
			suggestions := []string{}
			for _, compatibleInstance := range compatibleSpots {
				suggestions = append(suggestions, compatibleInstance.Type)
			}
			return ErrorAtLeastOneInstanceDistribution(*cc.InstanceType, suggestions[0], suggestions[1:]...)
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

	if strings.HasPrefix(instanceMetadata.Type, "inf") {
		return ErrorInstanceTypeNotSupported(instanceMetadata.Type)
	}

	if _, ok := awsutils.InstanceENIsAvailable[instanceMetadata.Type]; !ok {
		return ErrorInstanceTypeNotSupported(instanceMetadata.Type)
	}

	return nil
}

func CheckSpotInstanceCompatibility(target aws.InstanceMetadata, suggested aws.InstanceMetadata) error {
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

func CompatibleSpotInstances(awsClient *aws.Client, targetInstance aws.InstanceMetadata, maxPrice *float64, numInstances int) []aws.InstanceMetadata {
	compatibleInstances := []aws.InstanceMetadata{}
	instanceMap := aws.InstanceMetadatas[targetInstance.Region]
	availableInstances := []aws.InstanceMetadata{}

	for instanceType, instanceMetadata := range instanceMap {
		if instanceType == targetInstance.Type {
			continue
		}
		availableInstances = append(availableInstances, instanceMetadata)
	}

	sort.Slice(availableInstances, func(i, j int) bool {
		return availableInstances[i].Price < availableInstances[j].Price
	})

	for _, instanceMetadata := range availableInstances {
		if err := CheckCortexSupport(instanceMetadata); err != nil {
			continue
		}

		if err := CheckSpotInstanceCompatibility(targetInstance, instanceMetadata); err != nil {
			continue
		}

		spotInstancePrice, awsErr := awsClient.SpotInstancePrice(instanceMetadata.Region, instanceMetadata.Type)
		if awsErr == nil {
			if err := CheckSpotInstancePriceCompatibility(targetInstance, instanceMetadata, maxPrice, spotInstancePrice); err != nil {
				continue
			}
		}

		compatibleInstances = append(compatibleInstances, instanceMetadata)

		if len(compatibleInstances) == numInstances {
			break
		}
	}

	return compatibleInstances
}

func AutoGenerateSpotConfig(awsClient *aws.Client, spotConfig *SpotConfig, region string, instanceType string) error {
	chosenInstance := aws.InstanceMetadatas[region][instanceType]
	cleanedDistribution := []string{instanceType}
	for _, spotInstance := range spotConfig.InstanceDistribution {
		if spotInstance != instanceType {
			cleanedDistribution = append(cleanedDistribution, spotInstance)
		}
	}
	spotConfig.InstanceDistribution = cleanedDistribution

	if len(spotConfig.InstanceDistribution) == 1 {
		compatibleSpots := CompatibleSpotInstances(awsClient, chosenInstance, spotConfig.MaxPrice, _spotInstanceDistributionLength)
		if len(compatibleSpots) == 0 {
			return errors.Wrap(ErrorNoCompatibleSpotInstanceFound(chosenInstance.Type), InstanceTypeKey)
		}

		for _, instance := range compatibleSpots {
			spotConfig.InstanceDistribution = append(spotConfig.InstanceDistribution, instance.Type)
		}
	}

	if spotConfig.MaxPrice == nil {
		spotConfig.MaxPrice = &chosenInstance.Price
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

func (cc *Config) AutoFillSpot(awsClient *aws.Client) error {
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
		Region:       pointer.String("us-west-2"),
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

func RegionPrompt(clusterConfig *Config) error {
	defaults := applyPromptDefaults(*clusterConfig)
	regionPrompt := &cr.PromptValidation{
		SkipNonNilFields: true,
		PromptItemValidations: []*cr.PromptItemValidation{
			{
				StructField: "Region",
				PromptOpts: &prompt.Options{
					Prompt: RegionUserKey,
				},
				StringPtrValidation: &cr.StringPtrValidation{
					Validator: validateRegion,
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

func InstallPrompt(clusterConfig *Config, awsClient *aws.Client) error {
	defaults := applyPromptDefaults(*clusterConfig)
	accountID, _, err := awsClient.GetCachedAccountID()
	if err != nil {
		return err
	}
	bucketID := hash.String(accountID + *clusterConfig.Region)[:10]

	defaultBucket := clusterConfig.ClusterName + "-" + bucketID
	if len(defaultBucket) > 63 {
		defaultBucket = defaultBucket[:63]
	}
	if strings.HasSuffix(defaultBucket, "-") {
		defaultBucket = defaultBucket[:len(defaultBucket)-1]
	}

	remainingPrompts := &cr.PromptValidation{
		SkipNonNilFields:   true,
		SkipNonEmptyFields: true,
		PromptItemValidations: []*cr.PromptItemValidation{
			{
				StructField: "Bucket",
				PromptOpts: &prompt.Options{
					Prompt: BucketUserKey,
				},
				StringValidation: &cr.StringValidation{
					Default:   defaultBucket,
					MinLength: 3,
					MaxLength: 63,
				},
			},
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

	err = cr.ReadPrompt(clusterConfig, remainingPrompts)
	if err != nil {
		return err
	}

	return nil
}

func UpdatePromptValidation(skipPopulatedFields bool, userClusterConfig *Config) *cr.PromptValidation {
	defaults := applyPromptDefaults(*userClusterConfig)

	return &cr.PromptValidation{
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
				Validator: validateRegion,
				Default:   pointer.String("us-west-2"),
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
	items.Add(InstanceVolumeSizeUserKey, cc.InstanceVolumeSize)
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
	items.Add(TelemetryUserKey, cc.Telemetry)
	items.Add(ImageOperatorUserKey, cc.ImageOperator)
	items.Add(ImageManagerUserKey, cc.ImageManager)
	items.Add(ImageDownloaderUserKey, cc.ImageDownloader)
	items.Add(ImageRequestMonitorUserKey, cc.ImageRequestMonitor)
	items.Add(ImageClusterAutoscalerUserKey, cc.ImageClusterAutoscaler)
	items.Add(ImageMetricsServerUserKey, cc.ImageMetricsServer)
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
