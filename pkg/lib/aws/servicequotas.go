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
