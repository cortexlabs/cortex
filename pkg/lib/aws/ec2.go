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

package aws

import (
	"math"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
)

// Returns the minimum of all AZs in the region, 0 if unable to retreive spot price for any reason
func SpotInstancePrice(accessKeyID string, secretAccessKey string, region string, instanceType string) (float64, error) {
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(region),
		DisableSSL:  aws.Bool(false),
		Credentials: credentials.NewStaticCredentials(accessKeyID, secretAccessKey, ""),
	})
	if err != nil {
		return 0, errors.WithStack(err)
	}

	svc := ec2.New(sess)

	result, err := svc.DescribeSpotPriceHistory(&ec2.DescribeSpotPriceHistoryInput{
		InstanceTypes:       []*string{aws.String(instanceType)},
		ProductDescriptions: []*string{aws.String("Linux/UNIX")},
		StartTime:           aws.Time(time.Now()),
	})

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
