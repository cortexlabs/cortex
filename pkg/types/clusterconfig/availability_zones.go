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

package clusterconfig

import (
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
)

var _azBlacklist = strset.New("us-east-1e")

func (cc *Config) setAvailabilityZones(awsClient *aws.Client) error {
	var extraInstances []string
	if cc.Spot != nil && *cc.Spot && len(cc.SpotConfig.InstanceDistribution) >= 0 {
		for _, instanceType := range cc.SpotConfig.InstanceDistribution {
			if instanceType != *cc.InstanceType {
				extraInstances = append(extraInstances, instanceType)
			}
		}
	}

	if len(cc.AvailabilityZones) == 0 {
		if err := cc.setDefaultAvailabilityZones(awsClient, *cc.InstanceType, extraInstances...); err != nil {
			return err
		}
		return nil
	}

	if err := cc.validateUserAvailabilityZones(awsClient, *cc.InstanceType, extraInstances...); err != nil {
		return err
	}

	return nil
}

func (cc *Config) setDefaultAvailabilityZones(awsClient *aws.Client, instanceType string, instanceTypes ...string) error {
	zones, err := awsClient.ListSupportedAvailabilityZones(instanceType, instanceTypes...)
	if err != nil {
		// Try again without checking instance types
		zones, err = awsClient.ListAvailabilityZones()
		if err != nil {
			return nil // Let eksctl choose the availability zones
		}
	}

	zones.Subtract(_azBlacklist)

	if len(zones) < 2 {
		return ErrorNotEnoughDefaultSupportedZones(awsClient.Region, zones, instanceType, instanceTypes...)
	}

	// See https://github.com/weaveworks/eksctl/blob/master/pkg/eks/api.go
	if awsClient.Region == "us-east-1" {
		zones.Shrink(2)
	} else {
		zones.Shrink(3)
	}

	cc.AvailabilityZones = zones.SliceSorted()

	return nil
}

func (cc *Config) validateUserAvailabilityZones(awsClient *aws.Client, instanceType string, instanceTypes ...string) error {
	allZones, err := awsClient.ListAvailabilityZones()
	if err != nil {
		return nil // Skip validation
	}

	for _, userZone := range cc.AvailabilityZones {
		if !allZones.Has(userZone) {
			return ErrorInvalidAvailabilityZone(userZone, allZones)
		}
	}

	supportedZones, err := awsClient.ListSupportedAvailabilityZones(instanceType, instanceTypes...)
	if err != nil {
		// Skip validation instance-based validation
		supportedZones = strset.Difference(allZones, _azBlacklist)
	}

	for _, userZone := range cc.AvailabilityZones {
		if !supportedZones.Has(userZone) {
			return ErrorUnsupportedAvailabilityZone(userZone, instanceType, instanceTypes...)
		}
	}

	return nil
}
