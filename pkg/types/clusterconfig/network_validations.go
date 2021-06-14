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

package clusterconfig

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/servicequotas"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
)

const (
	_elasticIPsQuotaCode         = "L-0263D0A3"
	_internetGatewayQuotaCode    = "L-A4707A72"
	_natGatewayQuotaCode         = "L-FE5A380F"
	_vpcQuotaCode                = "L-F678F1CE"
	_securityGroupsQuotaCode     = "L-E79EC296"
	_securityGroupRulesQuotaCode = "L-0EA8095F"
)

func VerifyNetworkQuotas(
	awsClient *aws.Client,
	requiredInternetGateways int,
	natGatewayRequired bool,
	highlyAvailableNATGateway bool,
	requiredVPCs int,
	availabilityZones strset.Set,
	numNodeGroups int,
	longestCIDRWhiteList int) error {

	desiredQuotaCodes := strset.New(_elasticIPsQuotaCode, _internetGatewayQuotaCode, _natGatewayQuotaCode, _vpcQuotaCode, _securityGroupsQuotaCode, _securityGroupRulesQuotaCode)
	quotaCodeToValueMap := map[string]int{}

	err := awsClient.ServiceQuotas().ListServiceQuotasPages(
		&servicequotas.ListServiceQuotasInput{
			ServiceCode: pointer.String("ec2"),
		},
		func(page *servicequotas.ListServiceQuotasOutput, lastPage bool) bool {
			if page == nil {
				return false
			}
			for _, quota := range page.Quotas {
				if quota == nil || quota.QuotaCode == nil || quota.Value == nil {
					continue
				}
				if desiredQuotaCodes.Has(*quota.QuotaCode) {
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

	err = awsClient.ServiceQuotas().ListServiceQuotasPages(
		&servicequotas.ListServiceQuotasInput{
			ServiceCode: pointer.String("vpc"),
		},
		func(page *servicequotas.ListServiceQuotasOutput, lastPage bool) bool {
			if page == nil {
				return false
			}
			for _, quota := range page.Quotas {
				if quota == nil || quota.QuotaCode == nil || quota.Value == nil {
					continue
				}
				if desiredQuotaCodes.Has(*quota.QuotaCode) {
					quotaCodeToValueMap[*quota.QuotaCode] = int(*quota.Value)
				}
			}
			return true
		},
	)
	if err != nil {
		return errors.WithStack(err)
	}

	skippedValidations := ""
	defer func() {
		if len(skippedValidations) > 0 {
			fmt.Println(skippedValidations)
		}
	}()

	// check internet GW quota
	if requiredInternetGateways > 0 {
		if internetGatewayQuota, found := quotaCodeToValueMap[_internetGatewayQuotaCode]; found {
			err := awsClient.VerifyInternetGatewayQuota(internetGatewayQuota, requiredInternetGateways)
			if err != nil {
				return err
			}
		} else {
			skippedValidations += fmt.Sprintf("skipping internet gateway quota verification: unable to find internet gateway quota (%s)\n", _internetGatewayQuotaCode)
		}
	}

	if natGatewayRequired {
		if natGatewayQuota, found := quotaCodeToValueMap[_natGatewayQuotaCode]; found {
			err := awsClient.VerifyNATGatewayQuota(natGatewayQuota, availabilityZones, highlyAvailableNATGateway)
			if err != nil {
				return err
			}
		} else {
			skippedValidations += fmt.Sprintf("skipping nat gateway quota verification: unable to find nat gateway quota (%s)\n", _natGatewayQuotaCode)
		}
	}

	// check EIP quota
	if natGatewayRequired {
		if eipQuota, found := quotaCodeToValueMap[_elasticIPsQuotaCode]; found {
			err := awsClient.VerifyEIPQuota(eipQuota, availabilityZones, highlyAvailableNATGateway)
			if err != nil {
				return err
			}
		} else {
			skippedValidations += fmt.Sprintf("skipping elastic ip quota verification: unable to find elastic ip quota (%s)\n", _elasticIPsQuotaCode)
		}
	}

	if requiredVPCs > 0 {
		if vpcQuota, found := quotaCodeToValueMap[_vpcQuotaCode]; found {
			err := awsClient.VerifyVPCQuota(vpcQuota, requiredVPCs)
			if err != nil {
				return err
			}
		} else {
			skippedValidations += fmt.Sprintf("skipping vpc quota verification: unable to find vpc quota (%s)\n", _vpcQuotaCode)
		}
	}

	if securityGroupRulesQuota, found := quotaCodeToValueMap[_securityGroupRulesQuotaCode]; found {
		err := awsClient.VerifySecurityGroupRulesQuota(securityGroupRulesQuota, availabilityZones, numNodeGroups, longestCIDRWhiteList)
		if err != nil {
			return err
		}
	} else {
		skippedValidations += fmt.Sprintf("skipping security group rules quota verification: unable to find security group rules quota (%s)\n", _securityGroupRulesQuotaCode)
	}

	if securityGroupsQuota, found := quotaCodeToValueMap[_securityGroupsQuotaCode]; found {
		err := awsClient.VerifySecurityGroupQuota(securityGroupsQuota, numNodeGroups)
		if err != nil {
			return err
		}
	} else {
		skippedValidations += fmt.Sprintf("skipping security group quota verification: unable to find security group quota (%s)\n", _securityGroupsQuotaCode)
	}

	return nil
}
