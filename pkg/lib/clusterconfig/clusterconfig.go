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
	"github.com/cortexlabs/cortex/pkg/lib/table"
)

type ClusterConfig struct {
	InstanceType                        *string  `json:"instance_type" yaml:"instance_type"`
	MinInstances                        *int64   `json:"min_instances" yaml:"min_instances"`
	MaxInstances                        *int64   `json:"max_instances" yaml:"max_instances"`
	Spot                                *bool    `json:"spot" yaml:"spot"`
	InstanceDistribution                []string `json:"instance_distribution" yaml:"instance_distribution"`
	OnDemandBaseCapacity                *int64   `json:"on_demand_base_capacity" yaml:"on_demand_base_capacity"`
	OnDemandPercentageAboveBaseCapacity *int64   `json:"on_demand_percentage_above_base_capacity" yaml:"on_demand_percentage_above_base_capacity"`
	MaxPrice                            *float64 `json:"max_price" yaml:"max_price"`
	SpotInstancePools                   *int64   `json:"spot_instance_pools" yaml:"spot_instance_pools"`
	ClusterName                         string   `json:"cluster_name" yaml:"cluster_name"`
	Region                              string   `json:"region" yaml:"region"`
	Bucket                              string   `json:"bucket" yaml:"bucket"`
	LogGroup                            string   `json:"log_group" yaml:"log_group"`
	Telemetry                           bool     `json:"telemetry" yaml:"telemetry"`
	ImagePredictorServe                 string   `json:"image_predictor_serve" yaml:"image_predictor_serve"`
	ImagePredictorServeGPU              string   `json:"image_predictor_serve_gpu" yaml:"image_predictor_serve_gpu"`
	ImageTFServe                        string   `json:"image_tf_serve" yaml:"image_tf_serve"`
	ImageTFServeGPU                     string   `json:"image_tf_serve_gpu" yaml:"image_tf_serve_gpu"`
	ImageTFAPI                          string   `json:"image_tf_api" yaml:"image_tf_api"`
	ImageONNXServe                      string   `json:"image_onnx_serve" yaml:"image_onnx_serve"`
	ImageONNXServeGPU                   string   `json:"image_onnx_serve_gpu" yaml:"image_onnx_serve_gpu"`
	ImageOperator                       string   `json:"image_operator" yaml:"image_operator"`
	ImageManager                        string   `json:"image_manager" yaml:"image_manager"`
	ImageDownloader                     string   `json:"image_downloader" yaml:"image_downloader"`
	ImageClusterAutoscaler              string   `json:"image_cluster_autoscaler" yaml:"image_cluster_autoscaler"`
	ImageMetricsServer                  string   `json:"image_metrics_server" yaml:"image_metrics_server"`
	ImageNvidia                         string   `json:"image_nvidia" yaml:"image_nvidia"`
	ImageFluentd                        string   `json:"image_fluentd" yaml:"image_fluentd"`
	ImageStatsd                         string   `json:"image_statsd" yaml:"image_statsd"`
	ImageIstioProxy                     string   `json:"image_istio_proxy" yaml:"image_istio_proxy"`
	ImageIstioPilot                     string   `json:"image_istio_pilot" yaml:"image_istio_pilot"`
	ImageIstioCitadel                   string   `json:"image_istio_citadel" yaml:"image_istio_citadel"`
	ImageIstioGalley                    string   `json:"image_istio_galley" yaml:"image_istio_galley"`
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
			StructField:       "Spot",
			BoolPtrValidation: &cr.BoolPtrValidation{},
		},
		{
			StructField: "InstanceDistribution",
			StringListValidation: &cr.StringListValidation{
				DisallowDups: true,
				Validator:    validateInstanceDistribution,
			},
		},
		{
			StructField: "OnDemandBaseCapacity",
			Int64PtrValidation: &cr.Int64PtrValidation{
				GreaterThanOrEqualTo: pointer.Int64(0),
			},
		},
		{
			StructField: "OnDemandPercentageAboveBaseCapacity",
			Int64PtrValidation: &cr.Int64PtrValidation{
				GreaterThanOrEqualTo: pointer.Int64(1),
				LessThanOrEqualTo:    pointer.Int64(100),
			},
		},
		{
			StructField: "MaxPrice",
			Float64PtrValidation: &cr.Float64PtrValidation{
				GreaterThanOrEqualTo: pointer.Float64(0),
			},
		},
		{
			StructField: "SpotInstancePools",
			Int64PtrValidation: &cr.Int64PtrValidation{
				GreaterThanOrEqualTo: pointer.Int64(1),
				LessThanOrEqualTo:    pointer.Int64(20),
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
			StringValidation: &cr.StringValidation{
				Default: "us-west-2",
			},
		},
		{
			StructField: "Bucket",
			StringValidation: &cr.StringValidation{
				Default:    "",
				AllowEmpty: true,
			},
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

func (cc *ClusterConfig) Validate() error {
	if *cc.MinInstances > *cc.MaxInstances {
		return ErrorMinInstancesGreaterThanMax(*cc.MinInstances, *cc.MaxInstances)
	}
	chosenInstanceType := *cc.InstanceType
	if _, ok := aws.InstanceMetadatas[cc.Region][chosenInstanceType]; !ok {
		return ErrorInstanceTypeNotSupportedInRegion(chosenInstanceType, cc.Region)
	}

	if cc.Spot != nil && *cc.Spot {
		chosenInstance := aws.InstanceMetadatas[cc.Region][chosenInstanceType]
		compatibleSpots, err := CompatibleSpotInstances(chosenInstance)
		if err != nil {
			return err
		}

		compatibleInstanceCount := 0
		for _, instanceType := range cc.InstanceDistribution {
			if instanceType == chosenInstanceType {
				continue
			}
			if _, ok := aws.InstanceMetadatas[cc.Region][instanceType]; !ok {
				return errors.Wrap(ErrorInstanceTypeNotSupportedInRegion(instanceType, cc.Region), InstanceDistributionKey)
			}

			instanceMetadata := aws.InstanceMetadatas[cc.Region][chosenInstanceType]

			err := CheckSpotInstanceCompatibility(chosenInstance, instanceMetadata)
			if err != nil {
				return errors.Wrap(err, InstanceDistributionKey)
			}

			compatibleInstanceCount++
		}

		if compatibleInstanceCount == 0 {
			return ErrorAtLeastOneInstanceDistribution(chosenInstanceType, compatibleSpots[0].Type)
		}
	} else {
		if len(cc.InstanceDistribution) > 0 {
			return ErrorConfiguredWhenSpotIsNotEnabled(InstanceDistributionKey)
		}
		if cc.OnDemandBaseCapacity != nil {
			return ErrorConfiguredWhenSpotIsNotEnabled(OnDemandBaseCapacityKey)
		}
		if cc.OnDemandPercentageAboveBaseCapacity != nil {
			return ErrorConfiguredWhenSpotIsNotEnabled(OnDemandPercentageAboveBaseCapacityKey)
		}
		if cc.MaxPrice != nil {
			return ErrorConfiguredWhenSpotIsNotEnabled(InstanceDistributionKey)
		}
		if cc.SpotInstancePools != nil {
			return ErrorConfiguredWhenSpotIsNotEnabled(SpotInstancePoolsKey)
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

func CompatibleSpotInstances(targetInstance aws.InstanceMetadata) ([]aws.InstanceMetadata, error) {
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
	if len(compatibleInstances) == 0 {
		return compatibleInstances, ErrorNoCompatibleSpotInstanceFound(targetInstance.Type)
	}
	return compatibleInstances, nil
}

func (cc *ClusterConfig) AutoFill() error {
	if cc.Spot != nil && *cc.Spot {
		chosenInstance := aws.InstanceMetadatas[cc.Region][*cc.InstanceType]
		if len(cc.InstanceDistribution) == 0 {
			compatibleSpots, err := CompatibleSpotInstances(chosenInstance)
			if err != nil {
				return err
			}

			sort.Slice(compatibleSpots, func(i, j int) bool {
				return compatibleSpots[i].Price < compatibleSpots[j].Price
			})

			cc.InstanceDistribution = append(cc.InstanceDistribution, chosenInstance.Type)
			for _, instance := range compatibleSpots {
				cc.InstanceDistribution = append(cc.InstanceDistribution, instance.Type)
				if len(cc.InstanceDistribution) == 3 {
					break
				}
			}
		}

		if cc.MaxPrice == nil {
			cc.MaxPrice = &chosenInstance.Price
		}

		if cc.OnDemandBaseCapacity == nil {
			onDemand := *cc.MinInstances
			if onDemand == 0 {
				onDemand = 1
			}
			cc.OnDemandBaseCapacity = pointer.Int64(onDemand)
		}

		if cc.OnDemandPercentageAboveBaseCapacity == nil {
			cc.OnDemandPercentageAboveBaseCapacity = pointer.Int64(1)
		}

		if cc.SpotInstancePools == nil {
			cc.SpotInstancePools = pointer.Int64(2)
		}
	}

	return nil
}

func applyPromptDefaults(defaults *ClusterConfig) *ClusterConfig {
	if defaults == nil {
		defaults = &ClusterConfig{}
	}
	if defaults.InstanceType == nil {
		defaults.InstanceType = pointer.String("m5.large")
	}
	if defaults.MinInstances == nil {
		defaults.MinInstances = pointer.Int64(1)
	}
	if defaults.MaxInstances == nil {
		defaults.MaxInstances = pointer.Int64(5)
	}

	if defaults.Spot == nil {
		defaults.Spot = pointer.Bool(false)
	}
	return defaults
}

func InstallPromptValidation(defaults *ClusterConfig) *cr.PromptValidation {
	defaults = applyPromptDefaults(defaults)
	promptItemValidations := []*cr.PromptItemValidation{
		{
			StructField: "InstanceType",
			PromptOpts: &prompt.Options{
				Prompt: "AWS instance type",
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
				Prompt: "Min instances",
			},
			Int64PtrValidation: &cr.Int64PtrValidation{
				Required:    true,
				Default:     defaults.MinInstances,
				GreaterThan: pointer.Int64(0),
			},
		},
		{
			StructField: "MaxInstances",
			PromptOpts: &prompt.Options{
				Prompt: "Max instances",
			},
			Int64PtrValidation: &cr.Int64PtrValidation{
				Required:    true,
				Default:     defaults.MaxInstances,
				GreaterThan: pointer.Int64(0),
			},
		},
		{
			StructField: "Spot",
			PromptOpts: &prompt.Options{
				Prompt:     "Enable spot",
				DefaultStr: "n",
			},
			BoolPtrValidation: &cr.BoolPtrValidation{
				Required:  true,
				StrToBool: map[string]bool{"y": true, "n": false},
			},
		},
	}

	return &cr.PromptValidation{
		SkipPopulatedFields:   true,
		PromptItemValidations: promptItemValidations,
	}
}

func PromptValidation(skipPopulatedFields bool, promptInstanceType bool, defaults *ClusterConfig) *cr.PromptValidation {
	defaults = applyPromptDefaults(defaults)
	var promptItemValidations []*cr.PromptItemValidation

	if promptInstanceType {
		promptItemValidations = append(promptItemValidations,
			&cr.PromptItemValidation{
				StructField: "InstanceType",
				PromptOpts: &prompt.Options{
					Prompt: "AWS instance type",
				},
				StringPtrValidation: &cr.StringPtrValidation{
					Required:  true,
					Default:   defaults.InstanceType,
					Validator: validateInstanceType,
				},
			},
		)
	}

	promptItemValidations = append(promptItemValidations,
		&cr.PromptItemValidation{
			StructField: "MinInstances",
			PromptOpts: &prompt.Options{
				Prompt: "Min instances",
			},
			Int64PtrValidation: &cr.Int64PtrValidation{
				Required:    true,
				Default:     defaults.MinInstances,
				GreaterThan: pointer.Int64(0),
			},
		},
		&cr.PromptItemValidation{
			StructField: "MaxInstances",
			PromptOpts: &prompt.Options{
				Prompt: "Max instances",
			},
			Int64PtrValidation: &cr.Int64PtrValidation{
				Required:    true,
				Default:     defaults.MaxInstances,
				GreaterThan: pointer.Int64(0),
			},
		},
	)

	return &cr.PromptValidation{
		SkipPopulatedFields:   skipPopulatedFields,
		PromptItemValidations: promptItemValidations,
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

func (cc *ClusterConfig) SetBucket(awsAccessKeyID string, awsSecretAccessKey string) error {
	if cc.Bucket != "" {
		return nil
	}

	awsAccountID, validCreds, err := aws.AccountID(awsAccessKeyID, awsSecretAccessKey, cc.Region)
	if err != nil {
		return err
	}
	if !validCreds {
		return ErrorInvalidAWSCredentials()
	}

	cc.Bucket = "cortex-" + hash.String(awsAccountID)[:10]
	return nil
}

func (cc *InternalClusterConfig) UserFacingString() string {
	var items []table.KV

	items = append(items, table.KV{K: "cluster version", V: cc.APIVersion})
	items = append(items, table.KV{K: "instance type", V: *cc.InstanceType})
	items = append(items, table.KV{K: "min instances", V: *cc.MinInstances})
	items = append(items, table.KV{K: "max instances", V: *cc.MaxInstances})
	items = append(items, table.KV{K: "spot", V: *cc.Spot})
	if cc.Spot != nil && *cc.Spot {
		items = append(items, table.KV{K: "instance distribution", V: cc.InstanceDistribution})
		items = append(items, table.KV{K: "on demand base capacity", V: *cc.OnDemandBaseCapacity})
		items = append(items, table.KV{K: "on demand percentage above base capacity", V: *cc.OnDemandPercentageAboveBaseCapacity})
		items = append(items, table.KV{K: "max price", V: *cc.MaxPrice})
		items = append(items, table.KV{K: "spot instance pools", V: *cc.SpotInstancePools})

	}
	items = append(items, table.KV{K: "cluster name", V: cc.ClusterName})
	items = append(items, table.KV{K: "region", V: cc.Region})
	items = append(items, table.KV{K: "bucket", V: cc.Bucket})
	items = append(items, table.KV{K: "log group", V: cc.LogGroup})
	items = append(items, table.KV{K: "telemetry", V: cc.Telemetry})
	items = append(items, table.KV{K: "image_predictor_serve", V: cc.ImagePredictorServe})
	items = append(items, table.KV{K: "image_predictor_serve_gpu", V: cc.ImagePredictorServeGPU})
	items = append(items, table.KV{K: "image_tf_serve", V: cc.ImageTFServe})
	items = append(items, table.KV{K: "image_tf_serve_gpu", V: cc.ImageTFServeGPU})
	items = append(items, table.KV{K: "image_tf_api", V: cc.ImageTFAPI})
	items = append(items, table.KV{K: "image_onnx_serve", V: cc.ImageONNXServe})
	items = append(items, table.KV{K: "image_onnx_serve_gpu", V: cc.ImageONNXServeGPU})
	items = append(items, table.KV{K: "image_operator", V: cc.ImageOperator})
	items = append(items, table.KV{K: "image_manager", V: cc.ImageManager})
	items = append(items, table.KV{K: "image_downloader", V: cc.ImageDownloader})
	items = append(items, table.KV{K: "image_cluster_autoscaler", V: cc.ImageClusterAutoscaler})
	items = append(items, table.KV{K: "image_metrics_server", V: cc.ImageMetricsServer})
	items = append(items, table.KV{K: "image_nvidia", V: cc.ImageNvidia})
	items = append(items, table.KV{K: "image_fluentd", V: cc.ImageFluentd})
	items = append(items, table.KV{K: "image_statsd", V: cc.ImageStatsd})
	items = append(items, table.KV{K: "image_istio_proxy", V: cc.ImageIstioProxy})
	items = append(items, table.KV{K: "image_istio_pilot", V: cc.ImageIstioPilot})
	items = append(items, table.KV{K: "image_istio_citadel", V: cc.ImageIstioCitadel})
	items = append(items, table.KV{K: "image_istio_galley", V: cc.ImageIstioGalley})

	return table.AlignKeyValue(items, ":", 1)
}
