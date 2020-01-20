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
	"math"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
)

func (c *Client) SpotInstancePrice(region string, instanceType string) (float64, error) {
	result, err := c.EC2().DescribeSpotPriceHistory(&ec2.DescribeSpotPriceHistoryInput{
		InstanceTypes:       []*string{aws.String(instanceType)},
		ProductDescriptions: []*string{aws.String("Linux/UNIX")},
		StartTime:           aws.Time(time.Now()),
	})
	if err != nil {
		return 0, errors.WithStack(err)
	}

	min := math.MaxFloat64

	for _, spotPrice := range result.SpotPriceHistory {
		if spotPrice == nil {
			continue
		}

		price, ok := s.ParseFloat64(*spotPrice.SpotPrice)
		if !ok {
			continue
		}

		if price < min {
			min = price
		}
	}

	if min == math.MaxFloat64 {
		return 0, ErrorNoValidSpotPrices(instanceType, region)
	}

	if min <= 0 {
		return 0, ErrorNoValidSpotPrices(instanceType, region)
	}

	return min, nil
}

func (c *Client) GetAvailabilityZones() ([]string, error) {
	input := &ec2.DescribeAvailabilityZonesInput{}
	result, err := c.EC2().DescribeAvailabilityZones(input)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	availabilityZones := []string{}
	for _, az := range result.AvailabilityZones {
		if az.ZoneName != nil {
			availabilityZones = append(availabilityZones, *az.ZoneName)
		}
	}

	return availabilityZones, nil
}
