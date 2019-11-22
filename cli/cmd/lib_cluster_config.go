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

package cmd

import (
	"fmt"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/clusterconfig"
	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/prompt"
	"github.com/cortexlabs/cortex/pkg/lib/slices"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/table"
)

func readClusterConfigFile(clusterConfig *clusterconfig.ClusterConfig, path string) error {
	errs := cr.ParseYAMLFile(clusterConfig, clusterconfig.UserValidation, path)
	if errors.HasErrors(errs) {
		return errors.FirstError(errs...)
	}

	return nil
}

func getInstallClusterConfig() (*clusterconfig.ClusterConfig, *AWSCredentials, error) {
	awsCreds, err := getAWSCredentials(&flagClusterConfig)
	if err != nil {
		return nil, nil, err
	}

	clusterConfig := &clusterconfig.ClusterConfig{}

	err = clusterconfig.SetFileDefaults(clusterConfig)
	if err != nil {
		return nil, nil, err
	}

	if flagClusterConfig != "" {
		err := readClusterConfigFile(clusterConfig, flagClusterConfig)
		if err != nil {
			return nil, nil, err
		}
	}

	err = clusterconfig.InstallPrompt(clusterConfig, awsCreds.AWSAccessKeyID, awsCreds.AWSSecretAccessKey)
	if err != nil {
		return nil, nil, err
	}

	if clusterConfig.Spot != nil && *clusterConfig.Spot {
		clusterConfig.AutoFillSpot()
	}

	err = clusterConfig.Validate()
	if err != nil {
		return nil, nil, err
	}

	confirmClusterConfig(clusterConfig, awsCreds)

	return clusterConfig, awsCreds, nil
}

func getUpdateClusterConfig(awsCreds *AWSCredentials) (*clusterconfig.ClusterConfig, error) {
	cachedClusterConfig := &clusterconfig.ClusterConfig{}
	err := readClusterConfigFile(cachedClusterConfig, cachedClusterConfigPath)
	if err != nil {
		return nil, err
	}

	userClusterConfig := &clusterconfig.ClusterConfig{}
	err = clusterconfig.SetFileDefaults(userClusterConfig)
	if err != nil {
		return nil, err
	}

	if flagClusterConfig == "" {
		userClusterConfig = cachedClusterConfig
		err := cr.ReadPrompt(userClusterConfig, clusterconfig.UpdatePromptValidation(false, cachedClusterConfig))
		if err != nil {
			return nil, err
		}
	} else {
		err = readClusterConfigFile(userClusterConfig, flagClusterConfig)
		if err != nil {
			return nil, err
		}

		userClusterConfig.ClusterName = cachedClusterConfig.ClusterName
		userClusterConfig.Region = cachedClusterConfig.Region

		if userClusterConfig.Bucket == nil {
			userClusterConfig.Bucket = cachedClusterConfig.Bucket
		}

		if userClusterConfig.InstanceType != nil && *userClusterConfig.InstanceType != *cachedClusterConfig.InstanceType {
			return nil, ErrorConfigCannotBeChangedOnUpdate(clusterconfig.InstanceTypeKey, *cachedClusterConfig.InstanceType)
		}
		userClusterConfig.InstanceType = cachedClusterConfig.InstanceType

		if userClusterConfig.Spot != nil && *userClusterConfig.Spot != *cachedClusterConfig.Spot {
			return nil, ErrorConfigCannotBeChangedOnUpdate(clusterconfig.SpotKey, *cachedClusterConfig.Spot)
		}
		userClusterConfig.Spot = cachedClusterConfig.Spot

		if userClusterConfig.SpotConfig != nil && s.Obj(userClusterConfig.SpotConfig) != s.Obj(cachedClusterConfig.SpotConfig) {
			if cachedClusterConfig.SpotConfig == nil {
				return nil, clusterconfig.ErrorConfiguredWhenSpotIsNotEnabled(clusterconfig.SpotConfigKey)
			}

			if slices.StrSliceElementsMatch(userClusterConfig.SpotConfig.InstanceDistribution, cachedClusterConfig.SpotConfig.InstanceDistribution) {
				return nil, ErrorConfigCannotBeChangedOnUpdate(clusterconfig.InstanceDistributionKey, cachedClusterConfig.SpotConfig.InstanceDistribution)
			}

			if userClusterConfig.SpotConfig.OnDemandBaseCapacity != nil && *userClusterConfig.SpotConfig.OnDemandBaseCapacity != *cachedClusterConfig.SpotConfig.OnDemandBaseCapacity {
				return nil, ErrorConfigCannotBeChangedOnUpdate(clusterconfig.OnDemandBaseCapacityKey, *cachedClusterConfig.SpotConfig.OnDemandBaseCapacity)
			}

			if userClusterConfig.SpotConfig.OnDemandPercentageAboveBaseCapacity != nil && *userClusterConfig.SpotConfig.OnDemandPercentageAboveBaseCapacity != *cachedClusterConfig.SpotConfig.OnDemandPercentageAboveBaseCapacity {
				return nil, ErrorConfigCannotBeChangedOnUpdate(clusterconfig.OnDemandPercentageAboveBaseCapacityKey, *cachedClusterConfig.SpotConfig.OnDemandPercentageAboveBaseCapacity)
			}

			if userClusterConfig.SpotConfig.MaxPrice != nil && *userClusterConfig.SpotConfig.MaxPrice != *cachedClusterConfig.SpotConfig.MaxPrice {
				return nil, ErrorConfigCannotBeChangedOnUpdate(clusterconfig.MaxPriceKey, *cachedClusterConfig.SpotConfig.MaxPrice)
			}

			if userClusterConfig.SpotConfig.InstancePools != nil && *userClusterConfig.SpotConfig.InstancePools != *cachedClusterConfig.SpotConfig.InstancePools {
				return nil, ErrorConfigCannotBeChangedOnUpdate(clusterconfig.InstancePoolsKey, *cachedClusterConfig.SpotConfig.InstancePools)
			}
		}

		err := cr.ReadPrompt(userClusterConfig, clusterconfig.UpdatePromptValidation(true, cachedClusterConfig))
		if err != nil {
			return nil, err
		}
	}

	err = userClusterConfig.Validate()
	if err != nil {
		return nil, err
	}

	confirmClusterConfig(userClusterConfig, awsCreds)

	return userClusterConfig, nil
}

func confirmClusterConfig(clusterConfig *clusterconfig.ClusterConfig, awsCreds *AWSCredentials) {
	prevConfig, _ := clusterconfig.GetFileDefaults()

	var items []table.KV
	items = append(items, table.KV{K: "cluster name", V: clusterConfig.ClusterName})
	items = append(items, table.KV{K: "AWS access key ID", V: s.MaskString(awsCreds.AWSAccessKeyID, 4)})
	if awsCreds.CortexAWSAccessKeyID != awsCreds.AWSAccessKeyID {
		items = append(items, table.KV{K: "AWS access key ID", V: s.MaskString(awsCreds.CortexAWSAccessKeyID, 4) + " (cortex)"})
	}
	items = append(items, table.KV{K: clusterconfig.RegionUserFacingKey, V: clusterConfig.Region})
	items = append(items, table.KV{K: clusterconfig.BucketUserFacingKey, V: clusterConfig.Bucket})

	items = append(items, table.KV{K: clusterconfig.SpotUserFacingKey, V: s.YesNo(clusterConfig.Spot != nil && *clusterConfig.Spot)})
	items = append(items, table.KV{K: clusterconfig.InstanceTypeUserFacingKey, V: *clusterConfig.InstanceType})
	items = append(items, table.KV{K: clusterconfig.MinInstancesUserFacingKey, V: *clusterConfig.MinInstances})
	items = append(items, table.KV{K: clusterconfig.MaxInstancesUserFacingKey, V: *clusterConfig.MaxInstances})

	if clusterConfig.LogGroup != prevConfig.LogGroup {
		items = append(items, table.KV{K: clusterconfig.LogGroupUserFacingKey, V: clusterConfig.LogGroup})
	}
	if clusterConfig.Telemetry != prevConfig.Telemetry {
		items = append(items, table.KV{K: clusterconfig.TelemetryUserFacingKey, V: clusterConfig.Telemetry})
	}

	if clusterConfig.ImagePredictorServe != prevConfig.ImagePredictorServe {
		items = append(items, table.KV{K: clusterconfig.ImagePredictorServeUserFacingKey, V: clusterConfig.ImagePredictorServe})
	}
	if clusterConfig.ImagePredictorServeGPU != prevConfig.ImagePredictorServeGPU {
		items = append(items, table.KV{K: clusterconfig.ImagePredictorServeGPUUserFacingKey, V: clusterConfig.ImagePredictorServeGPU})
	}
	if clusterConfig.ImageTFServe != prevConfig.ImageTFServe {
		items = append(items, table.KV{K: clusterconfig.ImageTFServeUserFacingKey, V: clusterConfig.ImageTFServe})
	}
	if clusterConfig.ImageTFServeGPU != prevConfig.ImageTFServeGPU {
		items = append(items, table.KV{K: clusterconfig.ImageTFServeGPUUserFacingKey, V: clusterConfig.ImageTFServeGPU})
	}
	if clusterConfig.ImageTFAPI != prevConfig.ImageTFAPI {
		items = append(items, table.KV{K: clusterconfig.ImageTFAPIUserFacingKey, V: clusterConfig.ImageTFAPI})
	}
	if clusterConfig.ImageONNXServe != prevConfig.ImageONNXServe {
		items = append(items, table.KV{K: clusterconfig.ImageONNXServeUserFacingKey, V: clusterConfig.ImageONNXServe})
	}
	if clusterConfig.ImageONNXServeGPU != prevConfig.ImageONNXServeGPU {
		items = append(items, table.KV{K: clusterconfig.ImageONNXServeGPUUserFacingKey, V: clusterConfig.ImageONNXServeGPU})
	}
	if clusterConfig.ImageOperator != prevConfig.ImageOperator {
		items = append(items, table.KV{K: clusterconfig.ImageOperatorUserFacingKey, V: clusterConfig.ImageOperator})
	}
	if clusterConfig.ImageManager != prevConfig.ImageManager {
		items = append(items, table.KV{K: clusterconfig.ImageManagerUserFacingKey, V: clusterConfig.ImageManager})
	}
	if clusterConfig.ImageDownloader != prevConfig.ImageDownloader {
		items = append(items, table.KV{K: clusterconfig.ImageDownloaderUserFacingKey, V: clusterConfig.ImageDownloader})
	}
	if clusterConfig.ImageClusterAutoscaler != prevConfig.ImageClusterAutoscaler {
		items = append(items, table.KV{K: clusterconfig.ImageClusterAutoscalerUserFacingKey, V: clusterConfig.ImageClusterAutoscaler})
	}
	if clusterConfig.ImageMetricsServer != prevConfig.ImageMetricsServer {
		items = append(items, table.KV{K: clusterconfig.ImageMetricsServerUserFacingKey, V: clusterConfig.ImageMetricsServer})
	}
	if clusterConfig.ImageNvidia != prevConfig.ImageNvidia {
		items = append(items, table.KV{K: clusterconfig.ImageNvidiaUserFacingKey, V: clusterConfig.ImageNvidia})
	}
	if clusterConfig.ImageFluentd != prevConfig.ImageFluentd {
		items = append(items, table.KV{K: clusterconfig.ImageFluentdUserFacingKey, V: clusterConfig.ImageFluentd})
	}
	if clusterConfig.ImageStatsd != prevConfig.ImageStatsd {
		items = append(items, table.KV{K: clusterconfig.ImageStatsdUserFacingKey, V: clusterConfig.ImageStatsd})
	}
	if clusterConfig.ImageIstioProxy != prevConfig.ImageIstioProxy {
		items = append(items, table.KV{K: clusterconfig.ImageIstioProxyUserFacingKey, V: clusterConfig.ImageIstioProxy})
	}
	if clusterConfig.ImageIstioPilot != prevConfig.ImageIstioPilot {
		items = append(items, table.KV{K: clusterconfig.ImageIstioPilotUserFacingKey, V: clusterConfig.ImageIstioPilot})
	}
	if clusterConfig.ImageIstioCitadel != prevConfig.ImageIstioCitadel {
		items = append(items, table.KV{K: clusterconfig.ImageIstioCitadelUserFacingKey, V: clusterConfig.ImageIstioCitadel})
	}
	if clusterConfig.ImageIstioGalley != prevConfig.ImageIstioGalley {
		items = append(items, table.KV{K: clusterconfig.ImageIstioGalleyUserFacingKey, V: clusterConfig.ImageIstioGalley})
	}

	fmt.Println(table.AlignKeyValue(items, ":", 1) + "\n")

	exitMessage := fmt.Sprintf("Cluster configuration can be modified via the cluster config file; see https://www.cortex.dev/v/%s/cluster-management/config", consts.CortexVersion)
	prompt.YesOrExit("Is the configuration above correct?", exitMessage)
}
