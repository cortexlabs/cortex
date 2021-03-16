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
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
)

var _azBlacklist = strset.New("us-east-1e")

func (cc *Config) setAvailabilityZones(awsClient *aws.Client) error {
	if len(cc.AvailabilityZones) == 0 {
		if err := cc.setDefaultAvailabilityZones(awsClient); err != nil {
			return err
		}
		return nil
	}

	if err := cc.validateUserAvailabilityZones(awsClient); err != nil {
		return err
	}

	return nil
}

func (cc *Config) setDefaultAvailabilityZones(awsClient *aws.Client) error {
	instanceTypes := strset.New()
	for _, ng := range cc.NodeGroups {
		instanceTypes.Add(ng.InstanceType)
	}
	instanceTypesSlice := instanceTypes.Slice()

	zones, err := awsClient.ListSupportedAvailabilityZones(instanceTypesSlice[0], instanceTypesSlice[1:]...)
	if err != nil {
		// Try again without checking instance types
		zones, err = awsClient.ListAvailabilityZonesInRegion()
		if err != nil {
			return nil // Let eksctl choose the availability zones
		}
	}

	zones.Subtract(_azBlacklist)

	if len(zones) < 2 {
		return ErrorNotEnoughDefaultSupportedZones(awsClient.Region, zones, instanceTypesSlice[0], instanceTypesSlice[1:]...)
	}

	// See https://github.com/weaveworks/eksctl/blob/master/pkg/eks/api.go
	if awsClient.Region == "us-east-1" {
		zones.ShrinkSorted(2)
	} else {
		zones.ShrinkSorted(3)
	}

	cc.AvailabilityZones = zones.SliceSorted()

	return nil
}

func (cc *Config) validateUserAvailabilityZones(awsClient *aws.Client, extraInstances ...string) error {
	instanceTypes := strset.New()
	for _, ng := range cc.NodeGroups {
		instanceTypes.Add(ng.InstanceType)
	}
	instanceTypesSlice := instanceTypes.Slice()

	allZones, err := awsClient.ListAvailabilityZonesInRegion()
	if err != nil {
		return nil // Skip validation
	}

	for _, userZone := range cc.AvailabilityZones {
		if !allZones.Has(userZone) {
			return ErrorInvalidAvailabilityZone(userZone, allZones, awsClient.Region)
		}
	}

	supportedZones, err := awsClient.ListSupportedAvailabilityZones(instanceTypesSlice[0], instanceTypesSlice[1:]...)
	if err != nil {
		// Skip validation instance-based validation
		supportedZones = strset.Difference(allZones, _azBlacklist)
	}

	for _, userZone := range cc.AvailabilityZones {
		if !supportedZones.Has(userZone) {
			return ErrorUnsupportedAvailabilityZone(userZone, instanceTypesSlice[0], instanceTypesSlice[1:]...)
		}
	}

	return nil
}

func (cc *Config) validateSubnets(awsClient *aws.Client) error {
	if len(cc.Subnets) == 0 {
		return nil
	}

	allZones, err := awsClient.ListAvailabilityZonesInRegion()
	if err != nil {
		return nil // Skip validation
	}

	userZones := strset.New()

	for i, subnetConfig := range cc.Subnets {
		if !allZones.Has(subnetConfig.AvailabilityZone) {
			return errors.Wrap(ErrorInvalidAvailabilityZone(subnetConfig.AvailabilityZone, allZones, cc.Region), s.Index(i), AvailabilityZoneKey)
		}
		if userZones.Has(subnetConfig.AvailabilityZone) {
			return ErrorAvailabilityZoneSpecifiedTwice(subnetConfig.AvailabilityZone)
		}
		userZones.Add(subnetConfig.AvailabilityZone)
	}

	return nil
}
