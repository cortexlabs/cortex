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
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/servicequotas"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
)

var _standardInstanceFamilies = strset.New("a", "c", "d", "h", "i", "m", "r", "t", "z")
var _knownInstanceFamilies = strset.Union(_standardInstanceFamilies, strset.New("p", "g", "inf", "x", "f", "mac"))

const (
	_elasticIPsQuotaCode         = "L-0263D0A3"
	_internetGatewayQuotaCode    = "L-A4707A72"
	_natGatewayQuotaCode         = "L-FE5A380F"
	_vpcQuotaCode                = "L-F678F1CE"
	_securityGroupsQuotaCode     = "L-E79EC296"
	_securityGroupRulesQuotaCode = "L-0EA8095F"

	// 11 inbound rules
	_baseInboundRulesForNodeGroup = 11
	_inboundRulesPerAZ            = 8
	// ClusterSharedNodeSecurityGroup, ControlPlaneSecurityGroup, eks-cluster-sg-<cluster-name>, and operator security group
	_baseNumberOfSecurityGroups = 4
)

type InstanceTypeRequests struct {
	InstanceType              string
	RequiredOnDemandInstances int64
	RequiredSpotInstances     int64
}

type instanceClassRequest struct {
	InstanceClass string

	InstanceTypes strset.Set // the instance class (e.g. "standard") can have multiple instance types

	RequiredOnDemandCPUs int64
	RequiredSpotCPUs     int64

	OnDemandCPUQuota  *int64
	OnDemandQuotaCode string

	SpotCPUQuota  *int64
	SpotQuotaCode string
}

func (c *Client) VerifyInstanceQuota(instances []InstanceTypeRequests) error {
	instanceClassRequests := []instanceClassRequest{}
	for _, instance := range instances {
		if instance.RequiredOnDemandInstances == 0 && instance.RequiredSpotInstances == 0 {
			continue
		}

		parsedType, err := ParseInstanceType(instance.InstanceType)
		if err != nil {
			continue
		}

		// Allow the instance if we don't recognize the type
		if !_knownInstanceFamilies.Has(parsedType.Family) {
			continue
		}

		instanceClass := parsedType.Family
		if _standardInstanceFamilies.Has(parsedType.Family) {
			instanceClass = "standard"
		}

		cpusPerInstance := InstanceMetadatas[c.Region][instance.InstanceType].CPU

		instanceClassFound := false
		for idx, r := range instanceClassRequests {
			if r.InstanceClass == instanceClass {
				instanceClassRequests[idx].InstanceTypes.Add(instance.InstanceType)
				instanceClassRequests[idx].RequiredOnDemandCPUs += instance.RequiredOnDemandInstances * cpusPerInstance.Value()
				instanceClassRequests[idx].RequiredSpotCPUs += instance.RequiredSpotInstances * cpusPerInstance.Value()

				instanceClassFound = true
				break
			}
		}

		if !instanceClassFound {
			instanceClassRequests = append(instanceClassRequests, instanceClassRequest{
				InstanceClass:        instanceClass,
				InstanceTypes:        strset.New(instance.InstanceType),
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

				for idx, r := range instanceClassRequests {
					// quota is specified in number of vCPU permitted per family
					if strings.ToLower(*metricClass) == r.InstanceClass+"/ondemand" {
						instanceClassRequests[idx].OnDemandCPUQuota = pointer.Int64(int64(*quota.Value))
						instanceClassRequests[idx].OnDemandQuotaCode = *quota.QuotaCode
					} else if strings.ToLower(*metricClass) == r.InstanceClass+"/spot" {
						instanceClassRequests[idx].SpotCPUQuota = pointer.Int64(int64(*quota.Value))
						instanceClassRequests[idx].SpotQuotaCode = *quota.QuotaCode
					}
				}
			}
			return true
		},
	)
	if err != nil {
		return errors.WithStack(err)
	}

	for _, r := range instanceClassRequests {
		if r.OnDemandCPUQuota != nil && *r.OnDemandCPUQuota < r.RequiredOnDemandCPUs {
			return ErrorInsufficientInstanceQuota(r.InstanceTypes.Slice(), "on-demand", c.Region, r.RequiredOnDemandCPUs, *r.OnDemandCPUQuota, r.OnDemandQuotaCode)
		}
		if r.SpotCPUQuota != nil && *r.SpotCPUQuota < r.RequiredSpotCPUs {
			return ErrorInsufficientInstanceQuota(r.InstanceTypes.Slice(), "spot", c.Region, r.RequiredSpotCPUs, *r.SpotCPUQuota, r.SpotQuotaCode)
		}
	}

	return nil
}

func (c *Client) VerifyNetworkQuotas(
	requiredInternetGateways int,
	natGatewayRequired bool,
	highlyAvailableNATGateway bool,
	requiredVPCs int,
	availabilityZones strset.Set,
	numNodeGroups int,
	longestCIDRWhiteList int,
	clusterAlreadyExists bool,
) error {
	quotaCodeToValueMap := map[string]int{
		_elasticIPsQuotaCode:         0, // elastic IP quota code
		_internetGatewayQuotaCode:    0, // internet gw quota code
		_natGatewayQuotaCode:         0, // nat gw quota code
		_vpcQuotaCode:                0, // vpc quota code
		_securityGroupsQuotaCode:     0, // security groups quota code
		_securityGroupRulesQuotaCode: 0, // security group rules quota code
	}

	err := c.ServiceQuotas().ListServiceQuotasPages(
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

	if !clusterAlreadyExists {
		err = c.ServiceQuotas().ListServiceQuotasPages(
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
	}

	// check rules quota for nodegroup SGs
	requiredRulesForSG := requiredRulesForNodeGroupSecurityGroup(len(availabilityZones), longestCIDRWhiteList)
	if requiredRulesForSG > quotaCodeToValueMap[_securityGroupRulesQuotaCode] {
		additionalQuotaRequired := requiredRulesForSG - quotaCodeToValueMap[_securityGroupRulesQuotaCode]
		return ErrorSecurityGroupRulesExceeded(quotaCodeToValueMap[_securityGroupRulesQuotaCode], additionalQuotaRequired, c.Region)
	}

	// check rules quota for control plane SG
	requiredRulesForCPSG := requiredRulesForControlPlaneSecurityGroup(numNodeGroups)
	if requiredRulesForCPSG > quotaCodeToValueMap[_securityGroupRulesQuotaCode] {
		additionalQuotaRequired := requiredRulesForCPSG - quotaCodeToValueMap[_securityGroupRulesQuotaCode]
		return ErrorSecurityGroupRulesExceeded(quotaCodeToValueMap[_securityGroupRulesQuotaCode], additionalQuotaRequired, c.Region)
	}

	// check security groups quota
	requiredSecurityGroups := requiredSecurityGroups(numNodeGroups)
	sgs, err := c.DescribeSecurityGroups()
	if err != nil {
		return err
	}
	if quotaCodeToValueMap[_securityGroupsQuotaCode]-len(sgs)-requiredSecurityGroups < 0 {
		additionalQuotaRequired := len(sgs) + requiredSecurityGroups - quotaCodeToValueMap[_securityGroupsQuotaCode]
		return ErrorSecurityGroupLimitExceeded(quotaCodeToValueMap[_securityGroupsQuotaCode], additionalQuotaRequired, c.Region)

	}

	return nil
}

func (c *Client) VerifyNetworkQuotasOnConfigure(
	availabilityZones strset.Set,
	numNodeGroups int,
	longestCIDRWhiteList int) error {
	return c.VerifyNetworkQuotas(0, false, false, 0, availabilityZones, numNodeGroups, longestCIDRWhiteList, true)
}

func requiredRulesForNodeGroupSecurityGroup(numAZs, whitelistLength int) int {
	whitelistRuleCount := 0
	if whitelistLength == 1 {
		whitelistRuleCount = 1
	} else if whitelistLength > 1 {
		whitelistRuleCount = 1 + 5*(whitelistLength-1)
	}

	return _baseInboundRulesForNodeGroup + numAZs*_inboundRulesPerAZ + whitelistRuleCount
}

func requiredRulesForControlPlaneSecurityGroup(numNodeGroups int) int {
	// +1 for the operator node group
	// this is the number of outbound rules (there are half as many inbound rules, so that is not the limiting factor)
	return 2 * (numNodeGroups + 1)
}

func requiredSecurityGroups(numNodeGroups int) int {
	// each node group requires a security group
	return _baseNumberOfSecurityGroups + numNodeGroups
}
