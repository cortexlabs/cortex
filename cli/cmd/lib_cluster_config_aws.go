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

package cmd

import (
	"fmt"
	"path"
	"path/filepath"
	"regexp"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/console"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/maps"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/prompt"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/table"
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
	"github.com/cortexlabs/yaml"
)

var _cachedClusterConfigRegex = regexp.MustCompile(`^cluster_\S+\.yaml$`)

func cachedClusterConfigPath(clusterName string, region string) string {
	return filepath.Join(_localDir, fmt.Sprintf("cluster_%s_%s.yaml", clusterName, region))
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
	errs := cr.ParseYAMLFile(clusterConfig, clusterconfig.FullManagedValidation, filePath)
	if errors.HasError(errs) {
		return errors.FirstError(errs...)
	}

	return nil
}

func readUserClusterConfigFile(clusterConfig *clusterconfig.Config, filePath string) error {
	errs := cr.ParseYAMLFile(clusterConfig, clusterconfig.FullManagedValidation, filePath)
	if errors.HasError(errs) {
		return errors.Append(errors.FirstError(errs...), fmt.Sprintf("\n\ncluster configuration schema can be found at https://docs.cortex.dev/v/%s/", consts.CortexVersionMinor))
	}

	return nil
}

func getNewClusterAccessConfig(clusterConfigFile string) (*clusterconfig.AccessConfig, error) {
	accessConfig := &clusterconfig.AccessConfig{}

	errs := cr.ParseYAMLFile(accessConfig, clusterconfig.AccessValidation, clusterConfigFile)
	if errors.HasError(errs) {
		return nil, errors.Append(errors.FirstError(errs...), fmt.Sprintf("\n\ncluster configuration schema can be found at https://docs.cortex.dev/v/%s/", consts.CortexVersionMinor))
	}

	return accessConfig, nil
}

func getClusterAccessConfigWithCache() (*clusterconfig.AccessConfig, error) {
	accessConfig := &clusterconfig.AccessConfig{
		ImageManager: "quay.io/cortexlabs/manager:" + consts.CortexVersion,
	}

	cachedPaths := existingCachedClusterConfigPaths()
	if len(cachedPaths) == 1 {
		cachedAccessConfig := &clusterconfig.AccessConfig{}
		cr.ParseYAMLFile(cachedAccessConfig, clusterconfig.AccessValidation, cachedPaths[0])
		accessConfig.ClusterName = cachedAccessConfig.ClusterName
		accessConfig.Region = cachedAccessConfig.Region
	}

	if _flagClusterConfig != "" {
		errs := cr.ParseYAMLFile(accessConfig, clusterconfig.AccessValidation, _flagClusterConfig)
		if errors.HasError(errs) {
			return nil, errors.Append(errors.FirstError(errs...), fmt.Sprintf("\n\ncluster configuration schema can be found at https://docs.cortex.dev/v/%s/", consts.CortexVersionMinor))
		}
	}

	if _flagClusterName != "" {
		accessConfig.ClusterName = _flagClusterName
	}
	if _flagClusterRegion != "" {
		accessConfig.Region = _flagClusterRegion
	}

	if accessConfig.ClusterName == "" || accessConfig.Region == "" {
		return nil, ErrorClusterAccessConfigRequired()
	}
	return accessConfig, nil
}

func getInstallClusterConfig(awsClient *aws.Client, clusterConfigFile string, disallowPrompt bool) (*clusterconfig.Config, error) {
	clusterConfig := &clusterconfig.Config{}

	err := readUserClusterConfigFile(clusterConfig, clusterConfigFile)
	if err != nil {
		return nil, err
	}

	promptIfNotAdmin(awsClient, disallowPrompt)

	clusterConfig.Telemetry, err = readTelemetryConfig()
	if err != nil {
		return nil, err
	}

	err = clusterConfig.Validate(awsClient, false)
	if err != nil {
		err = errors.Append(err, fmt.Sprintf("\n\ncluster configuration schema can be found at https://docs.cortex.dev/v/%s/", consts.CortexVersionMinor))
		return nil, errors.Wrap(err, clusterConfigFile)
	}

	confirmInstallClusterConfig(clusterConfig, awsClient, disallowPrompt)

	return clusterConfig, nil
}

func getConfigureClusterConfig(cachedClusterConfig clusterconfig.Config, clusterConfigFile string, disallowPrompt bool) (*clusterconfig.Config, error) {
	userClusterConfig := &clusterconfig.Config{}
	var awsClient *aws.Client

	err := readUserClusterConfigFile(userClusterConfig, clusterConfigFile)
	if err != nil {
		return nil, err
	}

	userClusterConfig.ClusterName = cachedClusterConfig.ClusterName
	userClusterConfig.Region = cachedClusterConfig.Region
	awsClient, err = newAWSClient(userClusterConfig.Region)
	if err != nil {
		return nil, err
	}
	promptIfNotAdmin(awsClient, disallowPrompt)

	userClusterConfig.Telemetry, err = readTelemetryConfig()
	if err != nil {
		return nil, err
	}

	err = userClusterConfig.Validate(awsClient, true)
	if err != nil {
		err = errors.Append(err, fmt.Sprintf("\n\ncluster configuration schema can be found at https://docs.cortex.dev/v/%s/", consts.CortexVersionMinor))
		return nil, errors.Wrap(err, clusterConfigFile)
	}

	clusterConfigCopy, err := userClusterConfig.DeepCopy()
	if err != nil {
		return nil, err
	}

	cachedConfigCopy, err := cachedClusterConfig.DeepCopy()
	if err != nil {
		return nil, err
	}

	for idx := range clusterConfigCopy.NodeGroups {
		clusterConfigCopy.NodeGroups[idx].MinInstances = 0
		clusterConfigCopy.NodeGroups[idx].MaxInstances = 0
	}
	for idx := range cachedConfigCopy.NodeGroups {
		cachedConfigCopy.NodeGroups[idx].MinInstances = 0
		cachedConfigCopy.NodeGroups[idx].MaxInstances = 0
	}

	h1, err := clusterConfigCopy.Hash()
	if err != nil {
		return nil, err
	}
	h2, err := cachedConfigCopy.Hash()
	if err != nil {
		return nil, err
	}
	if h1 != h2 {
		return nil, clusterconfig.ErrorConfigCannotBeChangedOnUpdate()
	}

	yamlBytes, err := yaml.Marshal(userClusterConfig)
	if err != nil {
		return nil, err
	}
	yamlString := string(yamlBytes)

	fmt.Println(console.Bold("cluster config:"))
	fmt.Println(yamlString)

	if !disallowPrompt {
		exitMessage := fmt.Sprintf("cluster configuration can be modified via the cluster config file; see https://docs.cortex.dev/v/%s/ for more information", consts.CortexVersionMinor)
		prompt.YesOrExit(fmt.Sprintf("your cluster named \"%s\" in %s will be updated according to the configuration above, are you sure you want to continue?", userClusterConfig.ClusterName, userClusterConfig.Region), "", exitMessage)
	}

	return userClusterConfig, nil
}

func confirmInstallClusterConfig(clusterConfig *clusterconfig.Config, awsClient *aws.Client, disallowPrompt bool) {
	eksPrice := aws.EKSPrices[clusterConfig.Region]
	operatorInstancePrice := aws.InstanceMetadatas[clusterConfig.Region]["t3.medium"].Price
	operatorEBSPrice := aws.EBSMetadatas[clusterConfig.Region]["gp2"].PriceGB * 20 / 30 / 24
	metricsEBSPrice := aws.EBSMetadatas[clusterConfig.Region]["gp2"].PriceGB * 40 / 30 / 24
	nlbPrice := aws.NLBMetadatas[clusterConfig.Region].Price
	natUnitPrice := aws.NATMetadatas[clusterConfig.Region].Price

	var natTotalPrice float64
	if clusterConfig.NATGateway == clusterconfig.SingleNATGateway {
		natTotalPrice = natUnitPrice
	} else if clusterConfig.NATGateway == clusterconfig.HighlyAvailableNATGateway {
		natTotalPrice = natUnitPrice * float64(len(clusterConfig.AvailabilityZones))
	}

	headers := []table.Header{
		{Title: "aws resource"},
		{Title: "cost per hour"},
	}

	var rows [][]interface{}
	rows = append(rows, []interface{}{"1 eks cluster", s.DollarsMaxPrecision(eksPrice)})

	ngNameToSpotInstancesUsed := map[string]int{}
	fixedPrice := eksPrice + 2*operatorInstancePrice + operatorEBSPrice + metricsEBSPrice + 2*nlbPrice + natTotalPrice
	totalMinPrice := fixedPrice
	totalMaxPrice := fixedPrice
	for _, ng := range clusterConfig.NodeGroups {
		apiInstancePrice := aws.InstanceMetadatas[clusterConfig.Region][ng.InstanceType].Price
		apiEBSPrice := aws.EBSMetadatas[clusterConfig.Region][ng.InstanceVolumeType.String()].PriceGB * float64(ng.InstanceVolumeSize) / 30 / 24
		if ng.InstanceVolumeType.String() == "io1" && ng.InstanceVolumeIOPS != nil {
			apiEBSPrice += aws.EBSMetadatas[clusterConfig.Region][ng.InstanceVolumeType.String()].PriceIOPS * float64(*ng.InstanceVolumeIOPS) / 30 / 24
		}

		totalMinPrice += float64(ng.MinInstances) * (apiInstancePrice + apiEBSPrice)
		totalMaxPrice += float64(ng.MaxInstances) * (apiInstancePrice + apiEBSPrice)

		instanceStr := "instances"
		volumeStr := "volumes"
		if ng.MinInstances == 1 && ng.MaxInstances == 1 {
			instanceStr = "instance"
			volumeStr = "volume"
		}
		workerInstanceStr := fmt.Sprintf("nodegroup %s: %d - %d %s %s for your apis", ng.Name, ng.MinInstances, ng.MaxInstances, ng.InstanceType, instanceStr)
		ebsInstanceStr := fmt.Sprintf("nodegroup %s: %d - %d %dgb ebs %s for your apis", ng.Name, ng.MinInstances, ng.MaxInstances, ng.InstanceVolumeSize, volumeStr)
		if ng.MinInstances == ng.MaxInstances {
			workerInstanceStr = fmt.Sprintf("nodegroup %s: %d %s %s for your apis", ng.Name, ng.MinInstances, ng.InstanceType, instanceStr)
			ebsInstanceStr = fmt.Sprintf("nodegroup %s:%d %dgb ebs %s for your apis", ng.Name, ng.MinInstances, ng.InstanceVolumeSize, volumeStr)
		}

		workerPriceStr := s.DollarsMaxPrecision(apiInstancePrice) + " each"
		if ng.Spot {
			ngNameToSpotInstancesUsed[ng.Name]++
			spotPrice, err := awsClient.SpotInstancePrice(ng.InstanceType)
			workerPriceStr += " (spot pricing unavailable)"
			if err == nil && spotPrice != 0 {
				workerPriceStr = fmt.Sprintf("%s - %s each (varies based on spot price)", s.DollarsMaxPrecision(spotPrice), s.DollarsMaxPrecision(apiInstancePrice))
				totalMinPrice = fixedPrice + float64(ng.MinInstances)*(spotPrice+apiEBSPrice)
			}
		}

		rows = append(rows, []interface{}{workerInstanceStr, workerPriceStr})
		rows = append(rows, []interface{}{ebsInstanceStr, s.DollarsAndTenthsOfCents(apiEBSPrice) + " each"})
	}

	rows = append(rows, []interface{}{"2 t3.medium instances for cortex", s.DollarsMaxPrecision(operatorInstancePrice * 2)})
	rows = append(rows, []interface{}{"1 20gb ebs volume for the operator", s.DollarsAndTenthsOfCents(operatorEBSPrice)})
	rows = append(rows, []interface{}{"1 40gb ebs volume for prometheus", s.DollarsAndTenthsOfCents(metricsEBSPrice)})
	rows = append(rows, []interface{}{"2 network load balancers", s.DollarsMaxPrecision(nlbPrice) + " each"})

	if clusterConfig.NATGateway == clusterconfig.SingleNATGateway {
		rows = append(rows, []interface{}{"1 nat gateway", s.DollarsMaxPrecision(natUnitPrice)})
	} else if clusterConfig.NATGateway == clusterconfig.HighlyAvailableNATGateway {
		rows = append(rows, []interface{}{fmt.Sprintf("%d nat gateways", len(clusterConfig.AvailabilityZones)), s.DollarsMaxPrecision(natUnitPrice) + " each"})
	}

	items := table.Table{
		Headers: headers,
		Rows:    rows,
	}
	fmt.Println(items.MustFormat(&table.Opts{Sort: pointer.Bool(false)}))

	priceStr := s.DollarsAndCents(totalMaxPrice)
	suffix := ""
	if totalMinPrice != totalMaxPrice {
		priceStr = fmt.Sprintf("%s - %s", s.DollarsAndCents(totalMinPrice), s.DollarsAndCents(totalMaxPrice))
		if len(ngNameToSpotInstancesUsed) > 0 && len(ngNameToSpotInstancesUsed) < len(clusterConfig.NodeGroups) {
			suffix = " based on cluster size and spot instance pricing/availability"
		} else if len(ngNameToSpotInstancesUsed) == len(clusterConfig.NodeGroups) {
			suffix = " based on spot instance pricing/availability"
		} else if len(ngNameToSpotInstancesUsed) == 0 {
			suffix = " based on cluster size"
		}
	}

	fmt.Printf("your cluster will cost %s per hour%s\n\n", priceStr, suffix)

	privateSubnetMsg := ""
	if clusterConfig.SubnetVisibility == clusterconfig.PrivateSubnetVisibility {
		privateSubnetMsg = ", and will use private subnets for all EC2 instances"
	}
	fmt.Printf("cortex will also create an s3 bucket (%s) and a cloudwatch log group (%s)%s\n\n", clusterConfig.Bucket, clusterConfig.ClusterName, privateSubnetMsg)

	if clusterConfig.OperatorLoadBalancerScheme == clusterconfig.InternalLoadBalancerScheme {
		fmt.Print(fmt.Sprintf("warning: you've configured the operator load balancer to be internal; you must configure VPC Peering to connect your CLI to your cluster operator (see https://docs.cortex.dev/v/%s/)\n\n", consts.CortexVersionMinor))
	}

	if len(clusterConfig.Subnets) > 0 {
		fmt.Print("warning: you've configured your cluster to be installed in an existing VPC; if your cluster doesn't spin up or function as expected, please double-check your VPC configuration (here are the requirements: https://eksctl.io/usage/vpc-networking/#use-existing-vpc-other-custom-configuration)\n\n")
	}

	if len(clusterConfig.NodeGroups) > 1 && len(ngNameToSpotInstancesUsed) > 0 {
		fmt.Printf("warning: you've enabled spot instances for %s %s; spot instances are not guaranteed to be available so please take that into account for production clusters; see https://docs.cortex.dev/v/%s/ for more information\n\n", s.PluralS("nodegroup", len(ngNameToSpotInstancesUsed)), s.StrsAnd(maps.StrMapKeysInt(ngNameToSpotInstancesUsed)), consts.CortexVersionMinor)
	}

	if !disallowPrompt {
		exitMessage := fmt.Sprintf("cluster configuration can be modified via the cluster config file; see https://docs.cortex.dev/v/%s/ for more information", consts.CortexVersionMinor)
		prompt.YesOrExit("would you like to continue?", "", exitMessage)
	}
}
