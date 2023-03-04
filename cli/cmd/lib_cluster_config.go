/*
Copyright 2022 Cortex Labs, Inc.

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
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/maps"
	libmath "github.com/cortexlabs/cortex/pkg/lib/math"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/prompt"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/table"
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
	"github.com/cortexlabs/cortex/pkg/types/clusterstate"
)

var _cachedClusterConfigRegex = regexp.MustCompile(`^cluster_\S+\.yaml$`)

func getCachedClusterConfigPath(clusterName string, region string) string {
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
	errs := cr.ParseYAMLFile(clusterConfig, clusterconfig.FullConfigValidation, filePath)
	if errors.HasError(errs) {
		return errors.FirstError(errs...)
	}

	return nil
}

func readUserClusterConfigFile(clusterConfig *clusterconfig.Config, filePath string) error {
	errs := cr.ParseYAMLFile(clusterConfig, clusterconfig.FullConfigValidation, filePath)
	if errors.HasError(errs) {
		return errors.Append(errors.FirstError(errs...), fmt.Sprintf("\n\ncluster configuration schema can be found at https://docs.cortexlabs.com/v/%s/", consts.CortexVersionMinor))
	}

	return nil
}

func getNewClusterAccessConfig(clusterConfigFile string) (*clusterconfig.AccessConfig, error) {
	accessConfig := &clusterconfig.AccessConfig{}

	errs := cr.ParseYAMLFile(accessConfig, clusterconfig.AccessValidation, clusterConfigFile)
	if errors.HasError(errs) {
		return nil, errors.Append(errors.FirstError(errs...), fmt.Sprintf("\n\ncluster configuration schema can be found at https://docs.cortexlabs.com/v/%s/", consts.CortexVersionMinor))
	}

	return accessConfig, nil
}

func getClusterAccessConfigWithCache(hasClusterFlags bool) (*clusterconfig.AccessConfig, error) {
	accessConfig := &clusterconfig.AccessConfig{
		ImageManager: consts.DefaultRegistry() + "/manager:" + consts.CortexVersion,
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
			return nil, errors.Append(errors.FirstError(errs...), fmt.Sprintf("\n\ncluster configuration schema can be found at https://docs.cortexlabs.com/v/%s/", consts.CortexVersionMinor))
		}
	}

	if _flagClusterName != "" {
		accessConfig.ClusterName = _flagClusterName
	}
	if _flagClusterRegion != "" {
		accessConfig.Region = _flagClusterRegion
	}

	if accessConfig.ClusterName == "" || accessConfig.Region == "" {
		return nil, ErrorClusterAccessConfigRequired(hasClusterFlags)
	}
	return accessConfig, nil
}

func getInstallClusterConfig(awsClient *aws.Client, clusterConfigFile string) (*clusterconfig.Config, error) {
	clusterConfig := &clusterconfig.Config{}

	err := readUserClusterConfigFile(clusterConfig, clusterConfigFile)
	if err != nil {
		return nil, err
	}

	clusterConfig.Telemetry = isTelemetryEnabled()

	err = clusterConfig.ValidateOnInstall(awsClient)
	if err != nil {
		err = errors.Append(err, fmt.Sprintf("\n\ncluster configuration schema can be found at https://docs.cortexlabs.com/v/%s/", consts.CortexVersionMinor))
		return nil, errors.Wrap(err, clusterConfigFile)
	}

	return clusterConfig, nil
}

func getConfigureClusterConfig(awsClient *aws.Client, k8sClient *k8s.Client, stacks clusterstate.ClusterStacks, cachedClusterConfig clusterconfig.Config, newClusterConfigFile string) (*clusterconfig.Config, clusterconfig.ConfigureChanges, error) {
	newUserClusterConfig := &clusterconfig.Config{}

	err := readUserClusterConfigFile(newUserClusterConfig, newClusterConfigFile)
	if err != nil {
		return nil, clusterconfig.ConfigureChanges{}, err
	}

	newUserClusterConfig.Telemetry = isTelemetryEnabled()
	cachedClusterConfig.Telemetry = newUserClusterConfig.Telemetry

	configureChanges, err := newUserClusterConfig.ValidateOnConfigure(awsClient, k8sClient, cachedClusterConfig, stacks.NodeGroupsStacks)
	if err != nil {
		err = errors.Append(err, fmt.Sprintf("\n\ncluster configuration schema can be found at https://docs.cortexlabs.com/v/%s/", consts.CortexVersionMinor))
		return nil, clusterconfig.ConfigureChanges{}, errors.Wrap(err, newClusterConfigFile)
	}

	return newUserClusterConfig, configureChanges, nil
}

func confirmInstallClusterConfig(clusterConfig *clusterconfig.Config, awsClient *aws.Client, disallowPrompt bool) {
	eksPrice := aws.EKSPrices[clusterConfig.Region]
	operatorInstancePrice := aws.InstanceMetadatas[clusterConfig.Region]["t3.medium"].Price
	prometheusInstancePrice := aws.InstanceMetadatas[clusterConfig.Region][clusterConfig.PrometheusInstanceType].Price
	operatorEBSPrice := aws.EBSMetadatas[clusterConfig.Region]["gp3"].PriceGB * 20 / 30 / 24
	prometheusEBSPrice := aws.EBSMetadatas[clusterConfig.Region]["gp3"].PriceGB * 20 / 30 / 24
	metricsEBSPrice := aws.EBSMetadatas[clusterConfig.Region]["gp2"].PriceGB * (40 + 2) / 30 / 24
	nlbPrice := aws.NLBMetadatas[clusterConfig.Region].Price
	elbPrice := aws.ELBMetadatas[clusterConfig.Region].Price
	natUnitPrice := aws.NATMetadatas[clusterConfig.Region].Price

	var loadBalancersPrice float64
	usesELBForAPILoadBalancer := clusterConfig.APILoadBalancerType == clusterconfig.ELBLoadBalancerType
	if usesELBForAPILoadBalancer {
		loadBalancersPrice = nlbPrice + elbPrice
	} else {
		loadBalancersPrice = 2 * nlbPrice
	}

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
	fixedPrice := eksPrice + 2*(operatorInstancePrice+operatorEBSPrice) + prometheusInstancePrice + prometheusEBSPrice + metricsEBSPrice + loadBalancersPrice + natTotalPrice
	totalMinPrice := fixedPrice
	totalMaxPrice := fixedPrice
	for _, ng := range clusterConfig.NodeGroups {
		apiInstancePrice := aws.InstanceMetadatas[clusterConfig.Region][ng.InstanceType].Price
		apiEBSPrice := aws.EBSMetadatas[clusterConfig.Region][ng.InstanceVolumeType.String()].PriceGB * float64(ng.InstanceVolumeSize) / 30 / 24
		if ng.InstanceVolumeType == clusterconfig.IO1VolumeType && ng.InstanceVolumeIOPS != nil {
			apiEBSPrice += aws.EBSMetadatas[clusterConfig.Region][ng.InstanceVolumeType.String()].PriceIOPS * float64(*ng.InstanceVolumeIOPS) / 30 / 24
		}
		if ng.InstanceVolumeType == clusterconfig.GP3VolumeType && ng.InstanceVolumeIOPS != nil && ng.InstanceVolumeThroughput != nil {
			apiEBSPrice += libmath.MaxFloat64(0, (aws.EBSMetadatas[clusterConfig.Region][ng.InstanceVolumeType.String()].PriceIOPS-3000)*float64(*ng.InstanceVolumeIOPS)/30/24)
			apiEBSPrice += libmath.MaxFloat64(0, (aws.EBSMetadatas[clusterConfig.Region][ng.InstanceVolumeType.String()].PriceThroughput-125)*float64(*ng.InstanceVolumeThroughput)/30/24)
		}

		totalMaxPrice += float64(ng.MaxInstances) * (apiInstancePrice + apiEBSPrice)

		workerInstanceStr := fmt.Sprintf("nodegroup %s: %d-%d %s instances", ng.Name, ng.MinInstances, ng.MaxInstances, ng.InstanceType)
		if ng.MinInstances == ng.MaxInstances {
			workerInstanceStr = fmt.Sprintf("nodegroup %s: %d %s %s", ng.Name, ng.MinInstances, ng.InstanceType, s.PluralS("instance", ng.MinInstances))
		}

		workerPriceStr := s.DollarsAndTenthsOfCents(apiInstancePrice+apiEBSPrice) + " each"
		if ng.Spot {
			ngNameToSpotInstancesUsed[ng.Name]++
			spotPrice, err := awsClient.SpotInstancePrice(ng.InstanceType)
			workerPriceStr += " (spot pricing unavailable)"
			if err == nil && spotPrice != 0 {
				workerPriceStr = fmt.Sprintf("%s - %s each (varies based on spot price)", s.DollarsAndTenthsOfCents(spotPrice+apiEBSPrice), s.DollarsAndTenthsOfCents(apiInstancePrice+apiEBSPrice))
				if ng.MinInstances > *ng.SpotConfig.OnDemandBaseCapacity {
					totalMinPrice += float64(ng.MinInstances-*ng.SpotConfig.OnDemandBaseCapacity)*(spotPrice+apiEBSPrice)*float64(100-*ng.SpotConfig.OnDemandPercentageAboveBaseCapacity)/100 +
						float64(ng.MinInstances-*ng.SpotConfig.OnDemandBaseCapacity)*(apiInstancePrice+apiEBSPrice)*float64(*ng.SpotConfig.OnDemandPercentageAboveBaseCapacity)/100 +
						float64(*ng.SpotConfig.OnDemandBaseCapacity)*(apiInstancePrice+apiEBSPrice)
				} else {
					totalMinPrice += float64(ng.MinInstances) * (apiInstancePrice + apiEBSPrice)
				}
			} else {
				totalMinPrice += float64(ng.MinInstances) * (apiInstancePrice + apiEBSPrice)
			}
		} else {
			totalMinPrice += float64(ng.MinInstances) * (apiInstancePrice + apiEBSPrice)
		}

		rows = append(rows, []interface{}{workerInstanceStr, workerPriceStr})
	}

	operatorNodeGroupPrice := 2 * (operatorInstancePrice + operatorEBSPrice)
	prometheusNodeGroupPrice := prometheusInstancePrice + prometheusEBSPrice + metricsEBSPrice
	rows = append(rows, []interface{}{"2 t3.medium instances (cortex system)", s.DollarsAndTenthsOfCents(operatorNodeGroupPrice) + " total"})
	rows = append(rows, []interface{}{fmt.Sprintf("1 %s instance (prometheus)", clusterConfig.PrometheusInstanceType), s.DollarsAndTenthsOfCents(prometheusNodeGroupPrice)})
	if usesELBForAPILoadBalancer {
		rows = append(rows, []interface{}{"1 network load balancer", s.DollarsMaxPrecision(nlbPrice)})
		rows = append(rows, []interface{}{"1 classic load balancer", s.DollarsMaxPrecision(elbPrice)})
	} else {
		rows = append(rows, []interface{}{"2 network load balancers", s.DollarsMaxPrecision(loadBalancersPrice) + " total"})
	}

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
		fmt.Print(fmt.Sprintf("warning: you've configured the operator load balancer to be internal; you must configure VPC Peering to connect your CLI to your cluster operator (see https://docs.cortexlabs.com/v/%s/)\n\n", consts.CortexVersionMinor))
	}

	if len(clusterConfig.Subnets) > 0 {
		fmt.Print("warning: you've configured your cluster to be installed in an existing VPC; if your cluster doesn't spin up or function as expected, please double-check your VPC configuration (here are the requirements: https://eksctl.io/usage/vpc-networking/#use-existing-vpc-other-custom-configuration)\n\n")
	}

	if len(clusterConfig.NodeGroups) > 1 && len(ngNameToSpotInstancesUsed) > 0 {
		fmt.Printf("warning: you've enabled spot instances for %s %s; spot instances are not guaranteed to be available so please take that into account for production clusters; see https://docs.cortexlabs.com/v/%s/ for more information\n\n", s.PluralS("nodegroup", len(ngNameToSpotInstancesUsed)), s.StrsAnd(maps.StrMapKeysInt(ngNameToSpotInstancesUsed)), consts.CortexVersionMinor)
	}

	if !disallowPrompt {
		exitMessage := fmt.Sprintf("cluster configuration can be modified via the cluster config file; see https://docs.cortexlabs.com/v/%s/ for more information", consts.CortexVersionMinor)
		prompt.YesOrExit("would you like to continue?", "", exitMessage)
	}
}

func confirmConfigureClusterConfig(configureChanges clusterconfig.ConfigureChanges, oldCc, newCc clusterconfig.Config, disallowPrompt bool) {
	fmt.Printf("your %s cluster in region %s will be updated as follows:\n\n", newCc.ClusterName, newCc.Region)

	for _, fieldToUpdate := range configureChanges.FieldsToUpdate {
		fmt.Printf("￮ %s will be updated\n", fieldToUpdate)
	}

	for _, ngName := range configureChanges.NodeGroupsToUpdate {
		ngOld := oldCc.GetNodeGroupByName(ngName)
		ngNew := newCc.GetNodeGroupByName(ngName)
		fmt.Printf("￮ %s\n", ngNew.UpdatePlan(ngOld))
	}

	for _, ngName := range configureChanges.NodeGroupsToAdd {
		fmt.Printf("￮ nodegroup %s will be added\n", ngName)
	}

	for _, ngName := range configureChanges.NodeGroupsToRemove {
		fmt.Printf("￮ nodegroup %s will be removed\n", ngName)
	}

	// EKS node groups that don't appear in the old/new cluster config; this is unlikely but can happen
	for _, ngName := range configureChanges.GetGhostEKSNodeGroups() {
		fmt.Printf("￮ EKS nodegroup %s will be removed\n", ngName)
	}

	fmt.Println()

	if !disallowPrompt {
		exitMessage := fmt.Sprintf("cluster configuration can be modified via the cluster config file; see https://docs.cortexlabs.com/v/%s/ for more information", consts.CortexVersionMinor)
		prompt.YesOrExit(fmt.Sprintf("your cluster named \"%s\" in %s will be updated according to the configuration above, are you sure you want to continue?", newCc.ClusterName, newCc.Region), "", exitMessage)
	}
}
