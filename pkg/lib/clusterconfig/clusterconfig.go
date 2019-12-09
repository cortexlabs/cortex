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

import (
	"sort"
	"strings"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/hash"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/prompt"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/table"
)

type ClusterConfig struct {
	InstanceType           *string     `json:"instance_type" yaml:"instance_type"`
	MinInstances           *int64      `json:"min_instances" yaml:"min_instances"`
	MaxInstances           *int64      `json:"max_instances" yaml:"max_instances"`
	InstanceVolumeSize     int64       `json:"instance_volume_size" yaml:"instance_volume_size"`
	Spot                   *bool       `json:"spot" yaml:"spot"`
	SpotConfig             *SpotConfig `json:"spot_config" yaml:"spot_config"`
	ClusterName            string      `json:"cluster_name" yaml:"cluster_name"`
	Region                 *string     `json:"region" yaml:"region"`
	Bucket                 *string     `json:"bucket" yaml:"bucket"`
	LogGroup               string      `json:"log_group" yaml:"log_group"`
	Telemetry              bool        `json:"telemetry" yaml:"telemetry"`
	ImagePredictorServe    string      `json:"image_predictor_serve" yaml:"image_predictor_serve"`
	ImagePredictorServeGPU string      `json:"image_predictor_serve_gpu" yaml:"image_predictor_serve_gpu"`
	ImageTFServe           string      `json:"image_tf_serve" yaml:"image_tf_serve"`
	ImageTFServeGPU        string      `json:"image_tf_serve_gpu" yaml:"image_tf_serve_gpu"`
	ImageTFAPI             string      `json:"image_tf_api" yaml:"image_tf_api"`
	ImageONNXServe         string      `json:"image_onnx_serve" yaml:"image_onnx_serve"`
	ImageONNXServeGPU      string      `json:"image_onnx_serve_gpu" yaml:"image_onnx_serve_gpu"`
	ImageOperator          string      `json:"image_operator" yaml:"image_operator"`
	ImageManager           string      `json:"image_manager" yaml:"image_manager"`
	ImageDownloader        string      `json:"image_downloader" yaml:"image_downloader"`
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
}

type InternalClusterConfig struct {
	ClusterConfig

	// Populated by operator
	ID                string               `json:"id"`
	APIVersion        string               `json:"api_version"`
	OperatorInCluster bool                 `json:"operator_in_cluster"`
	InstanceMetadata  aws.InstanceMetadata `json:"instance_metadata"`
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
							LessThanOrEqualTo:    pointer.Int64(20),
							AllowExplicitNull:    true,
						},
					},
				},
			},
		},
		{
			StructField: "ClusterName",
			StringValidation: &cr.StringValidation{
				Default: "cortex",
			},
		},
		{
			StructField: "Region",
			StringPtrValidation: &cr.StringPtrValidation{
				AllowedValues: aws.EKSSupportedRegions.Slice(),
			},
		},
		{
			StructField:         "Bucket",
			StringPtrValidation: &cr.StringPtrValidation{},
		},
		{
			StructField: "LogGroup",
			StringValidation: &cr.StringValidation{
				Default: "cortex",
			},
		},
		{
			StructField: "Telemetry",
			BoolValidation: &cr.BoolValidation{
				Default: true,
			},
		},
		{
			StructField: "ImagePredictorServe",
			StringValidation: &cr.StringValidation{
				Default: "cortexlabs/predictor-serve:" + consts.CortexVersion,
			},
		},
		{
			StructField: "ImagePredictorServeGPU",
			StringValidation: &cr.StringValidation{
				Default: "cortexlabs/predictor-serve-gpu:" + consts.CortexVersion,
			},
		},
		{
			StructField: "ImageTFServe",
			StringValidation: &cr.StringValidation{
				Default: "cortexlabs/tf-serve:" + consts.CortexVersion,
			},
		},
		{
			StructField: "ImageTFServeGPU",
			StringValidation: &cr.StringValidation{
				Default: "cortexlabs/tf-serve-gpu:" + consts.CortexVersion,
			},
		},
		{
			StructField: "ImageTFAPI",
			StringValidation: &cr.StringValidation{
				Default: "cortexlabs/tf-api:" + consts.CortexVersion,
			},
		},
		{
			StructField: "ImageONNXServe",
			StringValidation: &cr.StringValidation{
				Default: "cortexlabs/onnx-serve:" + consts.CortexVersion,
			},
		},
		{
			StructField: "ImageONNXServeGPU",
			StringValidation: &cr.StringValidation{
				Default: "cortexlabs/onnx-serve-gpu:" + consts.CortexVersion,
			},
		},
		{
			StructField: "ImageOperator",
			StringValidation: &cr.StringValidation{
				Default: "cortexlabs/operator:" + consts.CortexVersion,
			},
		},
		{
			StructField: "ImageManager",
			StringValidation: &cr.StringValidation{
				Default: "cortexlabs/manager:" + consts.CortexVersion,
			},
		},
		{
			StructField: "ImageDownloader",
			StringValidation: &cr.StringValidation{
				Default: "cortexlabs/downloader:" + consts.CortexVersion,
			},
		},
		{
			StructField: "ImageClusterAutoscaler",
			StringValidation: &cr.StringValidation{
				Default: "cortexlabs/cluster-autoscaler:" + consts.CortexVersion,
			},
		},
		{
			StructField: "ImageMetricsServer",
			StringValidation: &cr.StringValidation{
				Default: "cortexlabs/metrics-server:" + consts.CortexVersion,
			},
		},
		{
			StructField: "ImageNvidia",
			StringValidation: &cr.StringValidation{
				Default: "cortexlabs/nvidia:" + consts.CortexVersion,
			},
		},
		{
			StructField: "ImageFluentd",
			StringValidation: &cr.StringValidation{
				Default: "cortexlabs/fluentd:" + consts.CortexVersion,
			},
		},
		{
			StructField: "ImageStatsd",
			StringValidation: &cr.StringValidation{
				Default: "cortexlabs/statsd:" + consts.CortexVersion,
			},
		},
		{
			StructField: "ImageIstioProxy",
			StringValidation: &cr.StringValidation{
				Default: "cortexlabs/istio-proxy:" + consts.CortexVersion,
			},
		},
		{
			StructField: "ImageIstioPilot",
			StringValidation: &cr.StringValidation{
				Default: "cortexlabs/istio-pilot:" + consts.CortexVersion,
			},
		},
		{
			StructField: "ImageIstioCitadel",
			StringValidation: &cr.StringValidation{
				Default: "cortexlabs/istio-citadel:" + consts.CortexVersion,
			},
		},
		{
			StructField: "ImageIstioGalley",
			StringValidation: &cr.StringValidation{
				Default: "cortexlabs/istio-galley:" + consts.CortexVersion,
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

func (cc *ClusterConfig) Validate(accessKeyID string, secretAccessKey string) error {
	if *cc.MinInstances > *cc.MaxInstances {
		return ErrorMinInstancesGreaterThanMax(*cc.MinInstances, *cc.MaxInstances)
	}

	if _, ok := aws.InstanceMetadatas[*cc.Region][*cc.InstanceType]; !ok {
		return errors.Wrap(ErrorInstanceTypeNotSupportedInRegion(*cc.InstanceType, *cc.Region), InstanceTypeKey)
	}

	err := aws.VerifyInstanceQuota(accessKeyID, secretAccessKey, *cc.InstanceType, *cc.Region)
	if err != nil {
		return err
	}

	if cc.Spot != nil && *cc.Spot {
		chosenInstance := aws.InstanceMetadatas[*cc.Region][*cc.InstanceType]
		compatibleSpots := CompatibleSpotInstances(chosenInstance)
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

			instanceMetadata := aws.InstanceMetadatas[*cc.Region][*cc.InstanceType]

			err := CheckSpotInstanceCompatibility(chosenInstance, instanceMetadata)
			if err != nil {
				return errors.Wrap(err, InstanceDistributionKey)
			}

			compatibleInstanceCount++
		}

		if compatibleInstanceCount == 0 {
			suggestions := []string{}
			compatibleSpots := CompatibleSpotInstances(chosenInstance)
			for _, compatibleInstance := range compatibleSpots {
				suggestions = append(suggestions, compatibleInstance.Type)
				if len(suggestions) == 3 {
					break
				}
			}
			return ErrorAtLeastOneInstanceDistribution(*cc.InstanceType, suggestions...)
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
		strings.HasSuffix(instanceMetadata.Type, "micro") ||
		strings.HasSuffix(instanceMetadata.Type, "small") {
		ErrorInstanceTypeTooSmall()
	}
	if instanceMetadata.GPU > 0 && !strings.HasPrefix(instanceMetadata.Type, "p2") && !strings.HasPrefix(instanceMetadata.Type, "p3") {
		return ErrorGPUInstanceTypeNotSupported(instanceMetadata.Type)
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

func CompatibleSpotInstances(targetInstance aws.InstanceMetadata) []aws.InstanceMetadata {
	compatibleInstances := []aws.InstanceMetadata{}
	instanceMap := aws.InstanceMetadatas[targetInstance.Region]
	for instanceType, instanceMetadata := range instanceMap {
		if instanceType == targetInstance.Type {
			continue
		}

		if err := CheckCortexSupport(instanceMetadata); err != nil {
			continue
		}

		if err := CheckSpotInstanceCompatibility(targetInstance, instanceMetadata); err != nil {
			continue
		}

		compatibleInstances = append(compatibleInstances, instanceMetadata)
	}

	sort.Slice(compatibleInstances, func(i, j int) bool {
		return compatibleInstances[i].Price < compatibleInstances[j].Price
	})
	return compatibleInstances
}

func AutoGenerateSpotConfig(spotConfig *SpotConfig, region string, instanceType string) error {
	chosenInstance := aws.InstanceMetadatas[region][instanceType]
	if len(spotConfig.InstanceDistribution) == 0 {
		spotConfig.InstanceDistribution = append(spotConfig.InstanceDistribution, chosenInstance.Type)

		compatibleSpots := CompatibleSpotInstances(chosenInstance)
		if len(compatibleSpots) == 0 {
			return errors.Wrap(ErrorNoCompatibleSpotInstanceFound(chosenInstance.Type), InstanceTypeKey)
		}

		for _, instance := range compatibleSpots {
			spotConfig.InstanceDistribution = append(spotConfig.InstanceDistribution, instance.Type)
			if len(spotConfig.InstanceDistribution) == 3 {
				break
			}
		}
	} else {
		instanceDistributionSet := strset.New(spotConfig.InstanceDistribution...)
		instanceDistributionSet.Remove(instanceType)
		spotConfig.InstanceDistribution = append([]string{instanceType}, instanceDistributionSet.Slice()...)
	}
	if spotConfig.MaxPrice == nil {
		spotConfig.MaxPrice = &chosenInstance.Price
	}

	if spotConfig.OnDemandBaseCapacity == nil {
		spotConfig.OnDemandBaseCapacity = pointer.Int64(0)
	}

	if spotConfig.OnDemandPercentageAboveBaseCapacity == nil {
		spotConfig.OnDemandPercentageAboveBaseCapacity = pointer.Int64(1)
	}

	if spotConfig.InstancePools == nil {
		spotConfig.InstancePools = pointer.Int64(2)
	}

	return nil
}

func (cc *ClusterConfig) AutoFillSpot() error {
	if cc.SpotConfig == nil {
		cc.SpotConfig = &SpotConfig{}
	}
	err := AutoGenerateSpotConfig(cc.SpotConfig, *cc.Region, *cc.InstanceType)
	if err != nil {
		return err
	}
	return nil
}

func applyPromptDefaults(defaults ClusterConfig) *ClusterConfig {
	defaultConfig := &ClusterConfig{
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

func InstallPrompt(clusterConfig *ClusterConfig, awsAccessKeyID string, awsSecretAccessKey string) error {
	defaults := applyPromptDefaults(*clusterConfig)

	regionPrompt := &cr.PromptValidation{
		SkipPopulatedFields: true,
		PromptItemValidations: []*cr.PromptItemValidation{
			{
				StructField: "Region",
				PromptOpts: &prompt.Options{
					Prompt: RegionUserFacingKey,
				},
				StringPtrValidation: &cr.StringPtrValidation{
					AllowedValues: aws.EKSSupportedRegions.Slice(),
					Default:       defaults.Region,
				},
			},
		},
	}
	err := cr.ReadPrompt(clusterConfig, regionPrompt)
	if err != nil {
		return err
	}

	awsAccountID, validCreds, err := aws.AccountID(awsAccessKeyID, awsSecretAccessKey, *clusterConfig.Region)
	if err != nil {
		return err
	}
	if !validCreds {
		return ErrorInvalidAWSCredentials()
	}

	defaultBucket := pointer.String("cortex-" + hash.String(awsAccountID)[:10])

	remainingPrompts := &cr.PromptValidation{
		SkipPopulatedFields: true,
		PromptItemValidations: []*cr.PromptItemValidation{
			{
				StructField: "Bucket",
				PromptOpts: &prompt.Options{
					Prompt: BucketUserFacingKey,
				},
				StringPtrValidation: &cr.StringPtrValidation{
					Default: defaultBucket,
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

func UpdatePromptValidation(skipPopulatedFields bool, userClusterConfig *ClusterConfig) *cr.PromptValidation {
	defaults := applyPromptDefaults(*userClusterConfig)

	return &cr.PromptValidation{
		SkipPopulatedFields: skipPopulatedFields,
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
func SetFileDefaults(cc *ClusterConfig) error {
	var emptyMap interface{} = map[interface{}]interface{}{}
	errs := cr.Struct(cc, emptyMap, UserValidation)
	if errors.HasErrors(errs) {
		return errors.FirstError(errs...)
	}
	return nil
}

// This does not set defaults for fields that are prompted from the user
func GetFileDefaults() (*ClusterConfig, error) {
	cc := &ClusterConfig{}
	err := SetFileDefaults(cc)
	if err != nil {
		return nil, err
	}

	return cc, nil
}

func (cc *InternalClusterConfig) UserFacingTable() []table.KV {
	var items []table.KV

	items = append(items, table.KV{K: APIVersionUserFacingKey, V: cc.APIVersion})
	items = append(items, cc.ClusterConfig.UserFacingTable()...)
	return items
}

func (cc *InternalClusterConfig) UserFacingString() string {
	return table.AlignKeyValue(cc.UserFacingTable(), ":", 1)
}

func (cc *ClusterConfig) UserFacingTable() []table.KV {
	var items []table.KV

	items = append(items, table.KV{K: ClusterNameUserFacingKey, V: cc.ClusterName})
	items = append(items, table.KV{K: RegionUserFacingKey, V: *cc.Region})
	items = append(items, table.KV{K: BucketUserFacingKey, V: *cc.Bucket})
	items = append(items, table.KV{K: InstanceTypeUserFacingKey, V: *cc.InstanceType})
	items = append(items, table.KV{K: MinInstancesUserFacingKey, V: *cc.MinInstances})
	items = append(items, table.KV{K: MaxInstancesUserFacingKey, V: *cc.MaxInstances})
	items = append(items, table.KV{K: InstanceVolumeSizeUserFacingKey, V: cc.InstanceVolumeSize})
	items = append(items, table.KV{K: SpotUserFacingKey, V: s.YesNo(*cc.Spot)})

	if cc.Spot != nil && *cc.Spot {
		items = append(items, table.KV{K: InstanceDistributionUserFacingKey, V: cc.SpotConfig.InstanceDistribution})
		items = append(items, table.KV{K: OnDemandBaseCapacityUserFacingKey, V: *cc.SpotConfig.OnDemandBaseCapacity})
		items = append(items, table.KV{K: OnDemandPercentageAboveBaseCapacityUserFacingKey, V: *cc.SpotConfig.OnDemandPercentageAboveBaseCapacity})
		items = append(items, table.KV{K: MaxPriceUserFacingKey, V: *cc.SpotConfig.MaxPrice})
		items = append(items, table.KV{K: InstancePoolsUserFacingKey, V: *cc.SpotConfig.InstancePools})
	}
	items = append(items, table.KV{K: LogGroupUserFacingKey, V: cc.LogGroup})
	items = append(items, table.KV{K: TelemetryUserFacingKey, V: cc.Telemetry})
	items = append(items, table.KV{K: ImagePredictorServeUserFacingKey, V: cc.ImagePredictorServe})
	items = append(items, table.KV{K: ImagePredictorServeGPUUserFacingKey, V: cc.ImagePredictorServeGPU})
	items = append(items, table.KV{K: ImageTFServeUserFacingKey, V: cc.ImageTFServe})
	items = append(items, table.KV{K: ImageTFServeGPUUserFacingKey, V: cc.ImageTFServeGPU})
	items = append(items, table.KV{K: ImageTFAPIUserFacingKey, V: cc.ImageTFAPI})
	items = append(items, table.KV{K: ImageONNXServeUserFacingKey, V: cc.ImageONNXServe})
	items = append(items, table.KV{K: ImageONNXServeGPUUserFacingKey, V: cc.ImageONNXServeGPU})
	items = append(items, table.KV{K: ImageOperatorUserFacingKey, V: cc.ImageOperator})
	items = append(items, table.KV{K: ImageManagerUserFacingKey, V: cc.ImageManager})
	items = append(items, table.KV{K: ImageDownloaderUserFacingKey, V: cc.ImageDownloader})
	items = append(items, table.KV{K: ImageClusterAutoscalerUserFacingKey, V: cc.ImageClusterAutoscaler})
	items = append(items, table.KV{K: ImageMetricsServerUserFacingKey, V: cc.ImageMetricsServer})
	items = append(items, table.KV{K: ImageNvidiaUserFacingKey, V: cc.ImageNvidia})
	items = append(items, table.KV{K: ImageFluentdUserFacingKey, V: cc.ImageFluentd})
	items = append(items, table.KV{K: ImageStatsdUserFacingKey, V: cc.ImageStatsd})
	items = append(items, table.KV{K: ImageIstioProxyUserFacingKey, V: cc.ImageIstioProxy})
	items = append(items, table.KV{K: ImageIstioPilotUserFacingKey, V: cc.ImageIstioPilot})
	items = append(items, table.KV{K: ImageIstioCitadelUserFacingKey, V: cc.ImageIstioCitadel})
	items = append(items, table.KV{K: ImageIstioGalleyUserFacingKey, V: cc.ImageIstioGalley})

	return items
}

func (cc *ClusterConfig) UserFacingString() string {
	return table.AlignKeyValue(cc.UserFacingTable(), ":", 1)
}
