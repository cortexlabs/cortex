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
	"path"
	"path/filepath"
	"regexp"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/prompt"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/table"
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
)

var _cachedClusterConfigRegex = regexp.MustCompile(`^cluster_\S+\.yaml$`)

func cachedClusterConfigPath(clusterName string, region string) string {
	return filepath.Join(_localDir, fmt.Sprintf("cluster_%s_%s.yaml", clusterName, region))
}

func mountedClusterConfigPath(clusterName string, region string) string {
	return filepath.Join("/.cortex", fmt.Sprintf("cluster_%s_%s.yaml", clusterName, region))
}

func existingCachedClusterConfigPaths() []string {
	paths, err := files.ListDir(_localDir, false)
	if err != nil {
		return nil
	}

	var matches []string
	for _, p := range paths {
		if _cachedClusterConfigRegex.MatchString(path.Base(p)) {
			matches = append(matches, p)
		}
	}

	return matches
}

func readCachedClusterConfigFile(clusterConfig *clusterconfig.Config, filePath string) error {
	errs := cr.ParseYAMLFile(clusterConfig, clusterconfig.Validation, filePath)
	if errors.HasError(errs) {
		return errors.FirstError(errs...)
	}

	return nil
}

func readUserClusterConfigFile(clusterConfig *clusterconfig.Config) error {
	errs := cr.ParseYAMLFile(clusterConfig, clusterconfig.UserValidation, _flagClusterConfig)
	if errors.HasError(errs) {
		return errors.FirstError(errs...)
	}

	return nil
}

func getClusterAccessConfig() (*clusterconfig.AccessConfig, error) {
	accessConfig, err := clusterconfig.DefaultAccessConfig()
	if err != nil {
		return nil, err
	}

	if _flagClusterConfig != "" {
		errs := cr.ParseYAMLFile(accessConfig, clusterconfig.AccessValidation, _flagClusterConfig)
		if errors.HasError(errs) {
			return nil, errors.FirstError(errs...)
		}
	}

	if accessConfig.ClusterName != nil && accessConfig.Region != nil {
		return accessConfig, nil
	}

	cachedPaths := existingCachedClusterConfigPaths()
	if len(cachedPaths) == 1 {
		cachedAccessConfig := &clusterconfig.AccessConfig{}
		cr.ParseYAMLFile(cachedAccessConfig, clusterconfig.AccessValidation, cachedPaths[0])
		if accessConfig.ClusterName == nil {
			accessConfig.ClusterName = cachedAccessConfig.ClusterName
		}
		if accessConfig.Region == nil {
			accessConfig.Region = cachedAccessConfig.Region
		}
	}

	err = cr.ReadPrompt(accessConfig, clusterconfig.AccessPromptValidation)
	if err != nil {
		return nil, err
	}

	return accessConfig, nil
}

func getInstallClusterConfig(awsCreds AWSCredentials) (*clusterconfig.Config, error) {
	clusterConfig := &clusterconfig.Config{}

	err := clusterconfig.SetDefaults(clusterConfig)
	if err != nil {
		return nil, err
	}

	if _flagClusterConfig != "" {
		err := readUserClusterConfigFile(clusterConfig)
		if err != nil {
			return nil, err
		}
	}

	err = clusterconfig.RegionPrompt(clusterConfig)
	if err != nil {
		return nil, err
	}

	awsClient, err := newAWSClient(*clusterConfig.Region, awsCreds)
	if err != nil {
		return nil, err
	}

	err = clusterconfig.InstallPrompt(clusterConfig, awsClient)
	if err != nil {
		return nil, err
	}

	clusterConfig.Telemetry, err = readTelemetryConfig()
	if err != nil {
		return nil, err
	}

	if clusterConfig.Spot != nil && *clusterConfig.Spot {
		clusterConfig.AutoFillSpot(awsClient)
	}

	err = clusterConfig.Validate(awsClient)
	if err != nil {
		if _flagClusterConfig != "" {
			err = errors.Wrap(err, _flagClusterConfig)
		}
		return nil, err
	}

	confirmInstallClusterConfig(clusterConfig, awsCreds, awsClient)

	return clusterConfig, nil
}

func getClusterUpdateConfig(cachedClusterConfig clusterconfig.Config, awsCreds AWSCredentials) (*clusterconfig.Config, error) {
	userClusterConfig := &clusterconfig.Config{}
	var awsClient *aws.Client

	if _flagClusterConfig == "" {
		userClusterConfig = &cachedClusterConfig
		err := cr.ReadPrompt(userClusterConfig, clusterconfig.UpdatePromptValidation(false, &cachedClusterConfig))
		if err != nil {
			return nil, err
		}
		awsClient, err = newAWSClient(*userClusterConfig.Region, awsCreds)
		if err != nil {
			return nil, err
		}
	} else {
		err := readUserClusterConfigFile(userClusterConfig)
		if err != nil {
			return nil, err
		}

		userClusterConfig.ClusterName = cachedClusterConfig.ClusterName
		userClusterConfig.Region = cachedClusterConfig.Region
		awsClient, err = newAWSClient(*userClusterConfig.Region, awsCreds)
		if err != nil {
			return nil, err
		}

		if userClusterConfig.Bucket == nil {
			userClusterConfig.Bucket = cachedClusterConfig.Bucket
		}

		if userClusterConfig.InstanceType != nil && *userClusterConfig.InstanceType != *cachedClusterConfig.InstanceType {
			return nil, ErrorConfigCannotBeChangedOnUpdate(clusterconfig.InstanceTypeKey, *cachedClusterConfig.InstanceType)
		}
		userClusterConfig.InstanceType = cachedClusterConfig.InstanceType

		if len(userClusterConfig.AvailabilityZones) > 0 && !strset.New(userClusterConfig.AvailabilityZones...).IsEqual(strset.New(cachedClusterConfig.AvailabilityZones...)) {
			return nil, ErrorConfigCannotBeChangedOnUpdate(clusterconfig.AvailabilityZonesKey, cachedClusterConfig.AvailabilityZones)
		}
		userClusterConfig.AvailabilityZones = cachedClusterConfig.AvailabilityZones

		if userClusterConfig.InstanceVolumeSize != cachedClusterConfig.InstanceVolumeSize {
			return nil, ErrorConfigCannotBeChangedOnUpdate(clusterconfig.InstanceVolumeSizeKey, cachedClusterConfig.InstanceVolumeSize)
		}
		userClusterConfig.InstanceVolumeSize = cachedClusterConfig.InstanceVolumeSize

		if userClusterConfig.Spot != nil && *userClusterConfig.Spot != *cachedClusterConfig.Spot {
			return nil, ErrorConfigCannotBeChangedOnUpdate(clusterconfig.SpotKey, *cachedClusterConfig.Spot)
		}
		userClusterConfig.Spot = cachedClusterConfig.Spot

		if userClusterConfig.Spot != nil && *userClusterConfig.Spot {
			err = userClusterConfig.AutoFillSpot(awsClient)
			if err != nil {
				return nil, err
			}
		}

		if userClusterConfig.SpotConfig != nil && s.Obj(userClusterConfig.SpotConfig) != s.Obj(cachedClusterConfig.SpotConfig) {
			if cachedClusterConfig.SpotConfig == nil {
				return nil, clusterconfig.ErrorConfiguredWhenSpotIsNotEnabled(clusterconfig.SpotConfigKey)
			}

			if !strset.New(userClusterConfig.SpotConfig.InstanceDistribution...).IsEqual(strset.New(cachedClusterConfig.SpotConfig.InstanceDistribution...)) {
				return nil, errors.Wrap(ErrorConfigCannotBeChangedOnUpdate(clusterconfig.InstanceDistributionKey, cachedClusterConfig.SpotConfig.InstanceDistribution), clusterconfig.SpotConfigKey)
			}

			if userClusterConfig.SpotConfig.OnDemandBaseCapacity != nil && *userClusterConfig.SpotConfig.OnDemandBaseCapacity != *cachedClusterConfig.SpotConfig.OnDemandBaseCapacity {
				return nil, errors.Wrap(ErrorConfigCannotBeChangedOnUpdate(clusterconfig.OnDemandBaseCapacityKey, *cachedClusterConfig.SpotConfig.OnDemandBaseCapacity), clusterconfig.SpotConfigKey)
			}

			if userClusterConfig.SpotConfig.OnDemandPercentageAboveBaseCapacity != nil && *userClusterConfig.SpotConfig.OnDemandPercentageAboveBaseCapacity != *cachedClusterConfig.SpotConfig.OnDemandPercentageAboveBaseCapacity {
				return nil, errors.Wrap(ErrorConfigCannotBeChangedOnUpdate(clusterconfig.OnDemandPercentageAboveBaseCapacityKey, *cachedClusterConfig.SpotConfig.OnDemandPercentageAboveBaseCapacity), clusterconfig.SpotConfigKey)
			}

			if userClusterConfig.SpotConfig.MaxPrice != nil && *userClusterConfig.SpotConfig.MaxPrice != *cachedClusterConfig.SpotConfig.MaxPrice {
				return nil, errors.Wrap(ErrorConfigCannotBeChangedOnUpdate(clusterconfig.MaxPriceKey, *cachedClusterConfig.SpotConfig.MaxPrice), clusterconfig.SpotConfigKey)
			}

			if userClusterConfig.SpotConfig.InstancePools != nil && *userClusterConfig.SpotConfig.InstancePools != *cachedClusterConfig.SpotConfig.InstancePools {
				return nil, errors.Wrap(ErrorConfigCannotBeChangedOnUpdate(clusterconfig.InstancePoolsKey, *cachedClusterConfig.SpotConfig.InstancePools), clusterconfig.SpotConfigKey)
			}
		}
		userClusterConfig.SpotConfig = cachedClusterConfig.SpotConfig

		err = cr.ReadPrompt(userClusterConfig, clusterconfig.UpdatePromptValidation(true, &cachedClusterConfig))
		if err != nil {
			return nil, err
		}
	}

	var err error
	userClusterConfig.Telemetry, err = readTelemetryConfig()
	if err != nil {
		return nil, err
	}

	err = userClusterConfig.Validate(awsClient)
	if err != nil {
		if _flagClusterConfig != "" {
			err = errors.Wrap(err, _flagClusterConfig)
		}
		return nil, err
	}

	confirmUpdateClusterConfig(*userClusterConfig, awsCreds, awsClient)

	return userClusterConfig, nil
}

func confirmInstallClusterConfig(clusterConfig *clusterconfig.Config, awsCreds AWSCredentials, awsClient *aws.Client) {
	var spotPrice float64
	if clusterConfig.Spot != nil && *clusterConfig.Spot {
		var err error
		spotPrice, err = awsClient.SpotInstancePrice(*clusterConfig.Region, *clusterConfig.InstanceType)
		if err != nil {
			spotPrice = 0
		}
	}

	operatorInstancePrice := aws.InstanceMetadatas[*clusterConfig.Region]["t3.medium"].Price
	operatorEBSPrice := aws.EBSMetadatas[*clusterConfig.Region].Price * 20 / 30 / 24
	elbPrice := aws.ELBMetadatas[*clusterConfig.Region].Price
	natPrice := aws.NATMetadatas[*clusterConfig.Region].Price

	fmt.Printf("cortex will use your %s aws access key id to provision the following resources in the %s region of your aws account:\n\n", s.MaskString(awsCreds.AWSAccessKeyID, 4), *clusterConfig.Region)
	fmt.Printf("￮ an s3 bucket named %s\n", *clusterConfig.Bucket)
	fmt.Printf("￮ a cloudwatch log group named %s\n", clusterConfig.LogGroup)
	fmt.Printf("￮ an eks cluster named %s ($0.20 per hour)\n", clusterConfig.ClusterName)
	fmt.Printf("￮ a t3.medium ec2 instance for the operator (%s per hour)\n", s.DollarsMaxPrecision(operatorInstancePrice))
	fmt.Printf("￮ a 20gb ebs volume for the operator (%s per hour)\n", s.DollarsAndTenthsOfCents(operatorEBSPrice))
	fmt.Printf("￮ an elb for the operator and an elb for apis (%s per hour each)\n", s.DollarsMaxPrecision(elbPrice))
	fmt.Printf("￮ a nat gateway (%s per hour)\n", s.DollarsMaxPrecision(natPrice))
	fmt.Println(workloadInstancesStr(clusterConfig, spotPrice))

	fmt.Println()

	apiInstancePrice := aws.InstanceMetadatas[*clusterConfig.Region][*clusterConfig.InstanceType].Price
	apiEBSPrice := aws.EBSMetadatas[*clusterConfig.Region].Price * float64(clusterConfig.InstanceVolumeSize) / 30 / 24
	fixedPrice := 0.20 + operatorInstancePrice + operatorEBSPrice + 2*elbPrice + natPrice
	totalMinPrice := fixedPrice + float64(*clusterConfig.MinInstances)*(apiInstancePrice+apiEBSPrice)
	totalMaxPrice := fixedPrice + float64(*clusterConfig.MaxInstances)*(apiInstancePrice+apiEBSPrice)

	spotSuffix := ""
	if clusterConfig.Spot != nil && *clusterConfig.Spot {
		spotSuffix = " (not accounting for spot instances)"
	}

	if *clusterConfig.MinInstances == *clusterConfig.MaxInstances {
		fmt.Printf("this cluster will cost %s per hour%s\n\n", s.DollarsAndCents(totalMaxPrice), spotSuffix)
	} else {
		fmt.Printf("this cluster will cost %s per hour if the minimum number of instances are running and %s per hour if the maximum number of instances are running%s\n\n", s.DollarsAndCents(totalMinPrice), s.DollarsAndCents(totalMaxPrice), spotSuffix)
	}

	if clusterConfig.Spot != nil && *clusterConfig.Spot {
		if *clusterConfig.SpotConfig.OnDemandBaseCapacity == 0 && *clusterConfig.SpotConfig.OnDemandPercentageAboveBaseCapacity == 0 {
			fmt.Printf("WARNING: you've disabled on-demand instances (%s=0 and %s=0); spot instances are not guaranteed to be available so please take that into account for production clusters; see https://cortex.dev/v/%s/cluster-management/spot-instances for more information\n", clusterconfig.OnDemandBaseCapacityKey, clusterconfig.OnDemandPercentageAboveBaseCapacityKey, consts.CortexVersionMinor)
		} else {
			fmt.Printf("WARNING: you've enabled spot instances; spot instances are not guaranteed to be available so please take that into account for production clusters; see https://cortex.dev/v/%s/cluster-management/spot-instances for more information\n", consts.CortexVersionMinor)
		}
		fmt.Println()
	}

	exitMessage := fmt.Sprintf("cluster configuration can be modified via the cluster config file; see https://cortex.dev/v/%s/cluster-management/config for more information", consts.CortexVersionMinor)
	prompt.YesOrExit("would you like to continue?", exitMessage)
}

func confirmUpdateClusterConfig(clusterConfig clusterconfig.Config, awsCreds AWSCredentials, awsClient *aws.Client) {
	fmt.Println(clusterConfigConfirmaionStr(clusterConfig, awsCreds, awsClient))

	exitMessage := fmt.Sprintf("cluster configuration can be modified via the cluster config file; see https://cortex.dev/v/%s/cluster-management/config for more information", consts.CortexVersionMinor)
	prompt.YesOrExit(fmt.Sprintf("your cluster (%s in %s) will be updated according to the configuration above, are you sure you want to continue?", clusterConfig.ClusterName, *clusterConfig.Region), exitMessage)
}

func clusterConfigConfirmaionStr(clusterConfig clusterconfig.Config, awsCreds AWSCredentials, awsClient *aws.Client) string {
	defaultConfig, _ := clusterconfig.GetDefaults()

	var items table.KeyValuePairs

	items.Add("aws access key id", s.MaskString(awsCreds.AWSAccessKeyID, 4))
	if awsCreds.CortexAWSAccessKeyID != awsCreds.AWSAccessKeyID {
		items.Add("aws access key id", s.MaskString(awsCreds.CortexAWSAccessKeyID, 4)+" (cortex)")
	}
	items.Add(clusterconfig.RegionUserKey, clusterConfig.Region)
	if len(clusterConfig.AvailabilityZones) > 0 {
		items.Add(clusterconfig.AvailabilityZonesUserKey, clusterConfig.AvailabilityZones)
	}
	items.Add(clusterconfig.BucketUserKey, clusterConfig.Bucket)
	items.Add(clusterconfig.ClusterNameUserKey, clusterConfig.ClusterName)
	if clusterConfig.LogGroup != defaultConfig.LogGroup {
		items.Add(clusterconfig.LogGroupUserKey, clusterConfig.LogGroup)
	}

	items.Add(clusterconfig.InstanceTypeUserKey, *clusterConfig.InstanceType)
	items.Add(clusterconfig.MinInstancesUserKey, *clusterConfig.MinInstances)
	items.Add(clusterconfig.MaxInstancesUserKey, *clusterConfig.MaxInstances)
	if clusterConfig.InstanceVolumeSize != defaultConfig.InstanceVolumeSize {
		items.Add(clusterconfig.InstanceVolumeSizeUserKey, clusterConfig.InstanceVolumeSize)
	}

	if clusterConfig.Spot != nil && *clusterConfig.Spot != *defaultConfig.Spot {
		items.Add(clusterconfig.SpotUserKey, s.YesNo(clusterConfig.Spot != nil && *clusterConfig.Spot))

		if clusterConfig.SpotConfig != nil {
			defaultSpotConfig := clusterconfig.SpotConfig{}
			clusterconfig.AutoGenerateSpotConfig(awsClient, &defaultSpotConfig, *clusterConfig.Region, *clusterConfig.InstanceType)

			if !strset.New(clusterConfig.SpotConfig.InstanceDistribution...).IsEqual(strset.New(defaultSpotConfig.InstanceDistribution...)) {
				items.Add(clusterconfig.InstanceDistributionUserKey, clusterConfig.SpotConfig.InstanceDistribution)
			}

			if *clusterConfig.SpotConfig.OnDemandBaseCapacity != *defaultSpotConfig.OnDemandBaseCapacity {
				items.Add(clusterconfig.OnDemandBaseCapacityUserKey, *clusterConfig.SpotConfig.OnDemandBaseCapacity)
			}

			if *clusterConfig.SpotConfig.OnDemandPercentageAboveBaseCapacity != *defaultSpotConfig.OnDemandPercentageAboveBaseCapacity {
				items.Add(clusterconfig.OnDemandPercentageAboveBaseCapacityUserKey, *clusterConfig.SpotConfig.OnDemandPercentageAboveBaseCapacity)
			}

			if *clusterConfig.SpotConfig.MaxPrice != *defaultSpotConfig.MaxPrice {
				items.Add(clusterconfig.MaxPriceUserKey, *clusterConfig.SpotConfig.MaxPrice)
			}

			if *clusterConfig.SpotConfig.InstancePools != *defaultSpotConfig.InstancePools {
				items.Add(clusterconfig.InstancePoolsUserKey, *clusterConfig.SpotConfig.InstancePools)
			}
		}
	}

	if clusterConfig.Telemetry != defaultConfig.Telemetry {
		items.Add(clusterconfig.TelemetryUserKey, clusterConfig.Telemetry)
	}

	if clusterConfig.ImagePythonServe != defaultConfig.ImagePythonServe {
		items.Add(clusterconfig.ImagePythonServeUserKey, clusterConfig.ImagePythonServe)
	}
	if clusterConfig.ImagePythonServeGPU != defaultConfig.ImagePythonServeGPU {
		items.Add(clusterconfig.ImagePythonServeGPUUserKey, clusterConfig.ImagePythonServeGPU)
	}
	if clusterConfig.ImageTFServe != defaultConfig.ImageTFServe {
		items.Add(clusterconfig.ImageTFServeUserKey, clusterConfig.ImageTFServe)
	}
	if clusterConfig.ImageTFServeGPU != defaultConfig.ImageTFServeGPU {
		items.Add(clusterconfig.ImageTFServeGPUUserKey, clusterConfig.ImageTFServeGPU)
	}
	if clusterConfig.ImageTFAPI != defaultConfig.ImageTFAPI {
		items.Add(clusterconfig.ImageTFAPIUserKey, clusterConfig.ImageTFAPI)
	}
	if clusterConfig.ImageONNXServe != defaultConfig.ImageONNXServe {
		items.Add(clusterconfig.ImageONNXServeUserKey, clusterConfig.ImageONNXServe)
	}
	if clusterConfig.ImageONNXServeGPU != defaultConfig.ImageONNXServeGPU {
		items.Add(clusterconfig.ImageONNXServeGPUUserKey, clusterConfig.ImageONNXServeGPU)
	}
	if clusterConfig.ImageOperator != defaultConfig.ImageOperator {
		items.Add(clusterconfig.ImageOperatorUserKey, clusterConfig.ImageOperator)
	}
	if clusterConfig.ImageManager != defaultConfig.ImageManager {
		items.Add(clusterconfig.ImageManagerUserKey, clusterConfig.ImageManager)
	}
	if clusterConfig.ImageDownloader != defaultConfig.ImageDownloader {
		items.Add(clusterconfig.ImageDownloaderUserKey, clusterConfig.ImageDownloader)
	}
	if clusterConfig.ImageClusterAutoscaler != defaultConfig.ImageClusterAutoscaler {
		items.Add(clusterconfig.ImageClusterAutoscalerUserKey, clusterConfig.ImageClusterAutoscaler)
	}
	if clusterConfig.ImageMetricsServer != defaultConfig.ImageMetricsServer {
		items.Add(clusterconfig.ImageMetricsServerUserKey, clusterConfig.ImageMetricsServer)
	}
	if clusterConfig.ImageNvidia != defaultConfig.ImageNvidia {
		items.Add(clusterconfig.ImageNvidiaUserKey, clusterConfig.ImageNvidia)
	}
	if clusterConfig.ImageFluentd != defaultConfig.ImageFluentd {
		items.Add(clusterconfig.ImageFluentdUserKey, clusterConfig.ImageFluentd)
	}
	if clusterConfig.ImageStatsd != defaultConfig.ImageStatsd {
		items.Add(clusterconfig.ImageStatsdUserKey, clusterConfig.ImageStatsd)
	}
	if clusterConfig.ImageIstioProxy != defaultConfig.ImageIstioProxy {
		items.Add(clusterconfig.ImageIstioProxyUserKey, clusterConfig.ImageIstioProxy)
	}
	if clusterConfig.ImageIstioPilot != defaultConfig.ImageIstioPilot {
		items.Add(clusterconfig.ImageIstioPilotUserKey, clusterConfig.ImageIstioPilot)
	}
	if clusterConfig.ImageIstioCitadel != defaultConfig.ImageIstioCitadel {
		items.Add(clusterconfig.ImageIstioCitadelUserKey, clusterConfig.ImageIstioCitadel)
	}
	if clusterConfig.ImageIstioGalley != defaultConfig.ImageIstioGalley {
		items.Add(clusterconfig.ImageIstioGalleyUserKey, clusterConfig.ImageIstioGalley)
	}

	return items.String()
}

func workloadInstancesStr(clusterConfig *clusterconfig.Config, spotPrice float64) string {
	instanceRangeStr := fmt.Sprintf("an autoscaling group of %d - %d", *clusterConfig.MinInstances, *clusterConfig.MaxInstances)
	volumeRangeStr := fmt.Sprintf("%d - %d", *clusterConfig.MinInstances, *clusterConfig.MaxInstances)
	if *clusterConfig.MinInstances == *clusterConfig.MaxInstances {
		instanceRangeStr = s.Int64(*clusterConfig.MinInstances)
		volumeRangeStr = s.Int64(*clusterConfig.MinInstances)
	}

	instancesStr := "instances"
	volumesStr := "volumes"
	if *clusterConfig.MinInstances == 1 && *clusterConfig.MaxInstances == 1 {
		instancesStr = "instance"
		volumesStr = "volume"
	}

	instanceTypeStr := *clusterConfig.InstanceType
	instancePrice := aws.InstanceMetadatas[*clusterConfig.Region][*clusterConfig.InstanceType].Price
	instancePriceStr := fmt.Sprintf("(%s per hour each)", s.DollarsMaxPrecision(instancePrice))

	if clusterConfig.Spot != nil && *clusterConfig.Spot {
		instanceTypeStr = s.StrsOr(clusterConfig.SpotConfig.InstanceDistribution)
		spotPriceStr := "spot pricing not available"
		if spotPrice != 0 {
			spotPriceStr = fmt.Sprintf("~%s per hour spot", s.DollarsMaxPrecision(spotPrice))
		}
		instancePriceStr = fmt.Sprintf("(%s: %s per hour on-demand, %s)", *clusterConfig.InstanceType, s.DollarsMaxPrecision(instancePrice), spotPriceStr)
	}

	ebsPrice := aws.EBSMetadatas[*clusterConfig.Region].Price * float64(clusterConfig.InstanceVolumeSize) / 30 / 24

	str := fmt.Sprintf("￮ %s %s ec2 %s for apis %s\n", instanceRangeStr, instanceTypeStr, instancesStr, instancePriceStr)
	str += fmt.Sprintf("￮ %s %dgb ebs %s, one for each api instance (%s per hour each)", volumeRangeStr, clusterConfig.InstanceVolumeSize, volumesStr, s.DollarsAndTenthsOfCents(ebsPrice))
	return str
}
