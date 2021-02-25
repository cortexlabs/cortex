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

package aws

import (
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/servicequotas"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
)

var _instanceCategoryRegex = regexp.MustCompile(`[a-zA-Z]+`)
var _standardInstanceCategories = strset.New("a", "c", "d", "h", "i", "m", "r", "t", "z")
var _knownInstanceCategories = strset.Union(_standardInstanceCategories, strset.New("p", "g", "inf", "x", "f"))

func (c *Client) VerifyInstanceQuota(instanceType string, requiredOnDemandInstances int64, requiredSpotInstances int64) error {
	if requiredOnDemandInstances == 0 && requiredSpotInstances == 0 {
		return nil
	}

	instanceCategory := _instanceCategoryRegex.FindString(instanceType)

	// Allow the instance if we don't recognize the type
	if !_knownInstanceCategories.Has(instanceCategory) {
		return nil
	}

	if _standardInstanceCategories.Has(instanceCategory) {
		instanceCategory = "standard"
	}

	var onDemandCPUQuota *int64
	var onDemandQuotaCode string
	var spotCPUQuota *int64
	var spotQuotaCode string
	err := c.ServiceQuotas().ListServiceQuotasPages(
		&servicequotas.ListServiceQuotasInput{
			ServiceCode: aws.String("ec2"),
		},
		func(page *servicequotas.ListServiceQuotasOutput, lastPage bool) bool {
			if page == nil {
				return false
			}
			for _, quota := range page.Quotas {
				if quota == nil || quota.UsageMetric == nil || len(quota.UsageMetric.MetricDimensions) == 0 {
					continue
				}

				metricClass, ok := quota.UsageMetric.MetricDimensions["Class"]
				if !ok || metricClass == nil || !(strings.HasSuffix(*metricClass, "/OnDemand") || strings.HasSuffix(*metricClass, "/Spot")) {
					continue
				}

				// quota is specified in number of vCPU permitted per family
				if strings.ToLower(*metricClass) == instanceCategory+"/ondemand" {
					onDemandCPUQuota = pointer.Int64(int64(*quota.Value))
					onDemandQuotaCode = *quota.QuotaCode
				} else if strings.ToLower(*metricClass) == instanceCategory+"/spot" {
					spotCPUQuota = pointer.Int64(int64(*quota.Value))
					spotQuotaCode = *quota.QuotaCode
				}

				if onDemandCPUQuota != nil && spotCPUQuota != nil {
					return false
				}
			}
			return true
		},
	)
	if err != nil {
		return errors.WithStack(err)
	}

	cpuPerInstance := InstanceMetadatas[c.Region][instanceType].CPU
	requiredOnDemandCPU := requiredOnDemandInstances * cpuPerInstance.Value()
	requiredSpotCPU := requiredSpotInstances * cpuPerInstance.Value()

	if onDemandCPUQuota != nil && *onDemandCPUQuota < requiredOnDemandCPU {
		return ErrorInsufficientInstanceQuota(instanceType, "on-demand", c.Region, requiredOnDemandInstances, cpuPerInstance.Value(), *onDemandCPUQuota, onDemandQuotaCode)
	}

	if spotCPUQuota != nil && *spotCPUQuota < requiredSpotCPU {
		return ErrorInsufficientInstanceQuota(instanceType, "spot", c.Region, requiredSpotInstances, cpuPerInstance.Value(), *spotCPUQuota, spotQuotaCode)
	}

	return nil
}

func (c *Client) VerifyNetworkQuotas(requiredInternetGateways int, requiredNATGatewaysPerAZ int, highlyAvailableNATGateway bool, requiredVPCs int) error {
	ec2QuotaCodeToValueMap := map[string]int{
		"L-0263D0A3": 0, // elastic IP quota code
	}
	vpcQuotaCodeToValueMap := map[string]int{
		"L-A4707A72": 0, // internet gw quota code
		"L-FE5A380F": 0, // nat gw quota code
		"L-F678F1CE": 0, // vpc quota code
	}

	err := c.ServiceQuotas().ListServiceQuotasPages(
		&servicequotas.ListServiceQuotasInput{
			ServiceCode: aws.String("ec2"),
		},
		func(page *servicequotas.ListServiceQuotasOutput, lastPage bool) bool {
			if page == nil {
				return false
			}
			for _, quota := range page.Quotas {
				if quota == nil || quota.QuotaCode == nil || quota.Value == nil {
					continue
				}
				if _, ok := ec2QuotaCodeToValueMap[*quota.QuotaCode]; ok {
					ec2QuotaCodeToValueMap[*quota.QuotaCode] += int(*quota.Value)
					return false
				}
			}
			return true
		},
	)
	if err != nil {
		return errors.WithStack(err)
	}

	err = c.ServiceQuotas().ListServiceQuotasPages(
		&servicequotas.ListServiceQuotasInput{
			ServiceCode: aws.String("vpc"),
		},
		func(page *servicequotas.ListServiceQuotasOutput, lastPage bool) bool {
			if page == nil {
				return false
			}
			for _, quota := range page.Quotas {
				if quota == nil || quota.QuotaCode == nil || quota.Value == nil {
					continue
				}
				if _, ok := vpcQuotaCodeToValueMap[*quota.QuotaCode]; ok {
					vpcQuotaCodeToValueMap[*quota.QuotaCode] += int(*quota.Value)
				}
			}
			return true
		},
	)
	if err != nil {
		return errors.WithStack(err)
	}

	// get EIP in use
	elasticIPsInUse, err := c.ListElasticIPs()
	if err != nil {
		return errors.WithStack(err)
	}

	// get IGW in use
	internetGatewaysInUse, err := c.ListInternetGateways()
	if err != nil {
		return errors.WithStack(err)
	}

	// get NAT GW in use per AZ
	gateways, err := c.DescribeNATGateways()
	if err != nil {
		return errors.WithStack(err)
	}
	subnets, err := c.DescribeSubnets()
	if err != nil {
		return errors.WithStack(err)
	}
	gatewaysSubnetIDs := SubnetIDsFromNATGateways(gateways)
	azToGatewaysInUse := make(map[string]int)
	for _, gatewaySubnetID := range gatewaysSubnetIDs {
		for _, subnet := range subnets {
			if subnet.SubnetId == nil || subnet.AvailabilityZone == nil {
				continue
			}
			if *subnet.SubnetId == gatewaySubnetID {
				if _, ok := azToGatewaysInUse[*subnet.AvailabilityZone]; !ok {
					azToGatewaysInUse[*subnet.AvailabilityZone] = 1
				} else {
					azToGatewaysInUse[*subnet.AvailabilityZone] += 1
				}
			}
		}
	}

	// get number of AZs
	azs, err := c.ListAvailabilityZonesInRegion()
	if err != nil {
		return errors.WithStack(err)
	}
	numberOfAZs := len(azs)

	// get VPC IDs
	vpcIDs := []string{}
	for _, subnet := range subnets {
		if subnet.SubnetId == nil || subnet.VpcId == nil {
			continue
		}
		vpcIDs = append(vpcIDs, *subnet.VpcId)
	}

	// check NAT GW quota
	numOfExhaustedNATGatewayAZs := 0
	greatestNATGatewayQuotaDeficit := 0
	for _, numActiveGatewaysOnAZ := range azToGatewaysInUse {
		azDeficit := vpcQuotaCodeToValueMap["L-FE5A380F"] - numActiveGatewaysOnAZ - requiredNATGatewaysPerAZ
		if azDeficit < 0 {
			numOfExhaustedNATGatewayAZs += 1
			if -azDeficit > greatestNATGatewayQuotaDeficit {
				greatestNATGatewayQuotaDeficit = -azDeficit
			}
		}
	}
	if highlyAvailableNATGateway && numOfExhaustedNATGatewayAZs > 0 {
		return ErrorNATGatewayLimitExceeded(vpcQuotaCodeToValueMap["L-FE5A380F"], greatestNATGatewayQuotaDeficit, c.Region)
	} else if !highlyAvailableNATGateway && numOfExhaustedNATGatewayAZs == numberOfAZs {
		return ErrorNATGatewayLimitExceeded(vpcQuotaCodeToValueMap["L-FE5A380F"], greatestNATGatewayQuotaDeficit, c.Region)
	}

	// check EIP quota
	var requiredElasticIPs int
	if requiredNATGatewaysPerAZ > 0 {
		if highlyAvailableNATGateway {
			requiredElasticIPs = numberOfAZs
		} else {
			requiredElasticIPs = 1
		}
	}
	if ec2QuotaCodeToValueMap["L-0263D0A3"]-len(elasticIPsInUse)-requiredElasticIPs < 0 {
		additionalQuotaRequired := -ec2QuotaCodeToValueMap["L-0263D0A3"] + len(elasticIPsInUse) + requiredElasticIPs
		return ErrorEIPLimitExceeded(ec2QuotaCodeToValueMap["L-0263D0A3"], additionalQuotaRequired, c.Region)
	}

	// check internet GW quota
	if vpcQuotaCodeToValueMap["L-A4707A72"]-len(internetGatewaysInUse)-requiredInternetGateways < 0 {
		additionalQuotaRequired := -vpcQuotaCodeToValueMap["L-A4707A72"] + len(internetGatewaysInUse) + requiredInternetGateways
		return ErrorInternetGatewayLimitExceeded(vpcQuotaCodeToValueMap["L-A4707A72"], additionalQuotaRequired, c.Region)
	}

	// check VPC quota
	if vpcQuotaCodeToValueMap["L-F678F1CE"]-len(vpcIDs)-requiredVPCs < 0 {
		additionalQuotaRequired := -vpcQuotaCodeToValueMap["L-F678F1CE"] + len(vpcIDs) + requiredVPCs
		return ErrorVPCLimitExceeded(vpcQuotaCodeToValueMap["L-F678F1CE"], additionalQuotaRequired, c.Region)
	}

	return nil
}
