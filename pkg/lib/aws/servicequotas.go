/*
Copyright 2020 Cortex Labs, Inc.

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

func (c *Client) VerifyInstanceQuota(instanceType string) error {
	instanceCategory := _instanceCategoryRegex.FindString(instanceType)

	// Allow the instance if we don't recognize the type
	if !_knownInstanceCategories.Has(instanceCategory) {
		return nil
	}

	if _standardInstanceCategories.Has(instanceCategory) {
		instanceCategory = "standard"
	}

	var cpuLimit *int
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
				if !ok || metricClass == nil || !strings.HasSuffix(*metricClass, "/OnDemand") {
					continue
				}

				if strings.ToLower(*metricClass) == instanceCategory+"/ondemand" {
					cpuLimit = pointer.Int(int(*quota.Value)) // quota is specified in number of vCPU permitted per family
					return false
				}
			}
			return true
		},
	)
	if err != nil {
		return errors.WithStack(err)
	}

	if cpuLimit != nil && *cpuLimit == 0 {
		return ErrorInstanceTypeLimitIsZero(instanceType, c.Region)
	}

	return nil
}
