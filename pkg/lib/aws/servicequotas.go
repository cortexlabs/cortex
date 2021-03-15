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

const (
	_elasticIPsQuotaCode      = "L-0263D0A3"
	_internetGatewayQuotaCode = "L-A4707A72"
	_natGatewayQuotaCode      = "L-FE5A380F"
	_vpcQuotaCode             = "L-F678F1CE"
)

type InstanceTypeRequests struct {
	InstanceType              string
	RequiredOnDemandInstances int64
	RequiredSpotInstances     int64
}

type instanceCategoryRequests struct {
	InstanceTypes []string

	InstanceCategory     string
	RequiredOnDemandCPUs int64
	RequiredSpotCPUs     int64

	OnDemandCPUQuota  *int64
	OnDemandQuotaCode string

	SpotCPUQuota  *int64
	SpotQuotaCode string
}

func (c *Client) VerifyInstanceQuota(instances []InstanceTypeRequests) error {
	instanceCategories := []instanceCategoryRequests{}
	for _, instance := range instances {
		if instance.RequiredOnDemandInstances == 0 && instance.RequiredSpotInstances == 0 {
			continue
		}

		instanceCategoryStr := _instanceCategoryRegex.FindString(instance.InstanceType)
		// Allow the instance if we don't recognize the type
		if !_knownInstanceCategories.Has(instanceCategoryStr) {
			continue
		}
		if _standardInstanceCategories.Has(instanceCategoryStr) {
			instanceCategoryStr = "standard"
		}

		cpusPerInstance := InstanceMetadatas[c.Region][instance.InstanceType].CPU

		categoryFound := false
		for idx, instanceCategory := range instanceCategories {
			if instanceCategory.InstanceCategory == instanceCategoryStr {
				instanceCategories[idx].InstanceTypes = append(instanceCategories[idx].InstanceTypes, instance.InstanceType)
				instanceCategories[idx].RequiredOnDemandCPUs += instance.RequiredOnDemandInstances * cpusPerInstance.Value()
				instanceCategories[idx].RequiredSpotCPUs += instance.RequiredSpotInstances * cpusPerInstance.Value()

				categoryFound = true
			}
		}

		if !categoryFound {
			instanceCategories = append(instanceCategories, instanceCategoryRequests{
				InstanceTypes:        []string{instance.InstanceType},
				InstanceCategory:     instanceCategoryStr,
				RequiredOnDemandCPUs: instance.RequiredOnDemandInstances * cpusPerInstance.Value(),
				RequiredSpotCPUs:     instance.RequiredSpotInstances * cpusPerInstance.Value(),
			})
		}
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
				if quota == nil || quota.UsageMetric == nil || len(quota.UsageMetric.MetricDimensions) == 0 {
					continue
				}

				metricClass, ok := quota.UsageMetric.MetricDimensions["Class"]
				if !ok || metricClass == nil || !(strings.HasSuffix(*metricClass, "/OnDemand") || strings.HasSuffix(*metricClass, "/Spot")) {
					continue
				}

				for idx, instanceCategory := range instanceCategories {
					// quota is specified in number of vCPU permitted per family
					if strings.ToLower(*metricClass) == instanceCategory.InstanceCategory+"/ondemand" {
						instanceCategories[idx].OnDemandCPUQuota = pointer.Int64(int64(*quota.Value))
						instanceCategories[idx].OnDemandQuotaCode = *quota.QuotaCode
					} else if strings.ToLower(*metricClass) == instanceCategory.InstanceCategory+"/spot" {
						instanceCategories[idx].SpotCPUQuota = pointer.Int64(int64(*quota.Value))
						instanceCategories[idx].SpotQuotaCode = *quota.QuotaCode
					}
				}
			}
			return true
		},
	)
	if err != nil {
		return errors.WithStack(err)
	}

	for _, ic := range instanceCategories {
		if ic.OnDemandCPUQuota != nil && *ic.OnDemandCPUQuota < ic.RequiredOnDemandCPUs {
			return ErrorInsufficientInstanceQuota(strset.FromSlice(ic.InstanceTypes).Slice(), "on-demand", c.Region, ic.RequiredOnDemandCPUs, *ic.OnDemandCPUQuota, ic.OnDemandQuotaCode)
		}
		if ic.SpotCPUQuota != nil && *ic.SpotCPUQuota < ic.RequiredSpotCPUs {
			return ErrorInsufficientInstanceQuota(strset.FromSlice(ic.InstanceTypes).Slice(), "spot", c.Region, ic.RequiredSpotCPUs, *ic.SpotCPUQuota, ic.SpotQuotaCode)
		}
	}

	return nil
}

func (c *Client) VerifyNetworkQuotas(requiredInternetGateways int, natGatewayRequired bool, highlyAvailableNATGateway bool, requiredVPCs int, availabilityZones strset.Set) error {
	quotaCodeToValueMap := map[string]int{
		_elasticIPsQuotaCode:      0, // elastic IP quota code
		_internetGatewayQuotaCode: 0, // internet gw quota code
		_natGatewayQuotaCode:      0, // nat gw quota code
		_vpcQuotaCode:             0, // vpc quota code
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
				if _, ok := quotaCodeToValueMap[*quota.QuotaCode]; ok {
					quotaCodeToValueMap[*quota.QuotaCode] = int(*quota.Value)
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
				if _, ok := quotaCodeToValueMap[*quota.QuotaCode]; ok {
					quotaCodeToValueMap[*quota.QuotaCode] = int(*quota.Value)
				}
			}
			return true
		},
	)
	if err != nil {
		return errors.WithStack(err)
	}

	// check internet GW quota
	if requiredInternetGateways > 0 {
		internetGatewaysInUse, err := c.ListInternetGateways()
		if err != nil {
			return err
		}
		if quotaCodeToValueMap[_internetGatewayQuotaCode]-len(internetGatewaysInUse)-requiredInternetGateways < 0 {
			additionalQuotaRequired := len(internetGatewaysInUse) + requiredInternetGateways - quotaCodeToValueMap[_internetGatewayQuotaCode]
			return ErrorInternetGatewayLimitExceeded(quotaCodeToValueMap[_internetGatewayQuotaCode], additionalQuotaRequired, c.Region)
		}
	}

	if natGatewayRequired {
		// get NAT GW in use per selected AZ
		natGateways, err := c.DescribeNATGateways()
		if err != nil {
			return err
		}
		subnets, err := c.DescribeSubnets()
		if err != nil {
			return err
		}
		azToGatewaysInUse := map[string]int{}
		for _, natGateway := range natGateways {
			if natGateway.SubnetId == nil {
				continue
			}
			for _, subnet := range subnets {
				if subnet.SubnetId == nil || subnet.AvailabilityZone == nil {
					continue
				}
				if !availabilityZones.Has(*subnet.AvailabilityZone) {
					continue
				}
				if *subnet.SubnetId == *natGateway.SubnetId {
					azToGatewaysInUse[*subnet.AvailabilityZone]++
				}
			}
		}
		// check NAT GW quota
		numOfExhaustedNATGatewayAZs := 0
		azsWithQuotaDeficit := []string{}
		for az, numActiveGatewaysOnAZ := range azToGatewaysInUse {
			// -1 comes from the NAT gateway we require per AZ
			azDeficit := quotaCodeToValueMap[_natGatewayQuotaCode] - numActiveGatewaysOnAZ - 1
			if azDeficit < 0 {
				numOfExhaustedNATGatewayAZs++
				azsWithQuotaDeficit = append(azsWithQuotaDeficit, az)
			}
		}
		if (highlyAvailableNATGateway && numOfExhaustedNATGatewayAZs > 0) || (!highlyAvailableNATGateway && numOfExhaustedNATGatewayAZs == len(availabilityZones)) {
			return ErrorNATGatewayLimitExceeded(quotaCodeToValueMap[_natGatewayQuotaCode], 1, azsWithQuotaDeficit, c.Region)
		}
	}

	// check EIP quota
	if natGatewayRequired {
		elasticIPsInUse, err := c.ListElasticIPs()
		if err != nil {
			return err
		}
		var requiredElasticIPs int
		if highlyAvailableNATGateway {
			requiredElasticIPs = len(availabilityZones)
		} else {
			requiredElasticIPs = 1
		}
		if quotaCodeToValueMap[_elasticIPsQuotaCode]-len(elasticIPsInUse)-requiredElasticIPs < 0 {
			additionalQuotaRequired := len(elasticIPsInUse) + requiredElasticIPs - quotaCodeToValueMap[_elasticIPsQuotaCode]
			return ErrorEIPLimitExceeded(quotaCodeToValueMap[_elasticIPsQuotaCode], additionalQuotaRequired, c.Region)
		}
	}

	// check VPC quota
	if requiredVPCs > 0 {
		vpcs, err := c.DescribeVpcs()
		if err != nil {
			return err
		}
		if quotaCodeToValueMap[_vpcQuotaCode]-len(vpcs)-requiredVPCs < 0 {
			additionalQuotaRequired := len(vpcs) + requiredVPCs - quotaCodeToValueMap[_vpcQuotaCode]
			return ErrorVPCLimitExceeded(quotaCodeToValueMap[_vpcQuotaCode], additionalQuotaRequired, c.Region)
		}
	}

	return nil
}
