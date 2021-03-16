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

package gcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"google.golang.org/api/compute/v1"
)

func (c *Client) IsProjectIDValid() (bool, error) {
	compClient, err := c.Compute()
	if err != nil {
		return false, err
	}

	_, err = compClient.Projects.Get(c.ProjectID).Do()
	if err != nil {
		errorMessage := fmt.Sprintf("The resource 'projects/%s' was not found", c.ProjectID)
		if IsErrCode(err, 404, &errorMessage) {
			return false, nil
		}
		return false, errors.WithStack(err)
	}

	return true, nil
}

func (c *Client) IsZoneValid(zone string) (bool, error) {
	compClient, err := c.Compute()
	if err != nil {
		return false, err
	}

	_, err = compClient.Zones.Get(c.ProjectID, zone).Do()
	if err != nil {
		errorMessage := fmt.Sprintf("The resource 'projects/%s/zones/%s' was not found", c.ProjectID, zone)
		if IsErrCode(err, 404, &errorMessage) {
			return false, nil
		}
		return false, errors.WithStack(err)
	}

	return true, nil
}

func (c *Client) IsInstanceTypeAvailable(instanceType string, zone string) (bool, error) {
	compClient, err := c.Compute()
	if err != nil {
		return false, err
	}

	_, err = compClient.MachineTypes.Get(c.ProjectID, zone, instanceType).Do()
	if err != nil {
		errorMessage := fmt.Sprintf("The resource 'projects/%s/zones/%s/machineTypes/%s' was not found", c.ProjectID, zone, instanceType)
		if IsErrCode(err, 404, &errorMessage) {
			return false, nil
		}
		return false, errors.WithStack(err)
	}

	return true, nil
}

func (c *Client) IsAcceleratorTypeAvailable(acceleratorType string, zone string) (bool, error) {
	compClient, err := c.Compute()
	if err != nil {
		return false, err
	}

	_, err = compClient.AcceleratorTypes.Get(c.ProjectID, zone, acceleratorType).Do()
	if err != nil {
		errorMessage := fmt.Sprintf("The resource 'projects/%s/zones/%s/acceleratorTypes/%s' was not found", c.ProjectID, zone, acceleratorType)
		if IsErrCode(err, 404, &errorMessage) {
			return false, nil
		}
		return false, errors.WithStack(err)
	}

	return true, nil
}

func (c *Client) GetAvailableZones() ([]string, error) {
	compClient, err := c.Compute()
	if err != nil {
		return nil, err
	}

	availableZones := strset.New()

	err = compClient.Zones.List(c.ProjectID).Pages(context.Background(), func(zonesList *compute.ZoneList) error {
		if zonesList != nil {
			for _, zone := range zonesList.Items {
				if zone == nil {
					continue
				}
				availableZones.Add(zone.Name)
			}
		}
		return nil
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return availableZones.SliceSorted(), nil
}

func (c *Client) GetAvailableInstanceTypes(zone string) ([]string, error) {
	compClient, err := c.Compute()
	if err != nil {
		return nil, err
	}

	availableInstanceTypes := strset.New()

	err = compClient.MachineTypes.List(c.ProjectID, zone).Pages(context.Background(), func(instanceTypeList *compute.MachineTypeList) error {
		if instanceTypeList != nil {
			for _, machineType := range instanceTypeList.Items {
				if machineType == nil {
					continue
				}
				availableInstanceTypes.Add(machineType.Name)
			}
		}
		return nil
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return availableInstanceTypes.SliceSorted(), nil
}

func (c *Client) GetAvailableInstanceTypesForAllZones() ([]string, error) {
	compClient, err := c.Compute()
	if err != nil {
		return nil, err
	}

	availableInstanceTypes := strset.New()

	err = compClient.MachineTypes.AggregatedList(c.ProjectID).Pages(context.Background(), func(instanceTypeAggregatedList *compute.MachineTypeAggregatedList) error {
		if instanceTypeAggregatedList != nil {
			for _, scopedMachineTypes := range instanceTypeAggregatedList.Items {
				for _, machineType := range scopedMachineTypes.MachineTypes {
					if machineType == nil {
						continue
					}
					availableInstanceTypes.Add(machineType.Name)
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return availableInstanceTypes.SliceSorted(), nil
}

func (c *Client) GetInstanceTypesMetadata(zone string) ([]compute.MachineType, error) {
	compClient, err := c.Compute()
	if err != nil {
		return nil, err
	}

	machineTypesList := make([]compute.MachineType, 0)
	err = compClient.MachineTypes.List(c.ProjectID, zone).Pages(context.Background(), func(instanceTypeList *compute.MachineTypeList) error {
		if instanceTypeList != nil {
			for _, machineType := range instanceTypeList.Items {
				if machineType == nil {
					continue
				}
				machineTypesList = append(machineTypesList, *machineType)
			}
		}
		return nil
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return machineTypesList, nil
}

func (c *Client) GetInstanceTypesMetadataForAllZones() ([]compute.MachineType, error) {
	compClient, err := c.Compute()
	if err != nil {
		return nil, err
	}

	machineTypesList := make([]compute.MachineType, 0)
	err = compClient.MachineTypes.AggregatedList(c.ProjectID).Pages(context.Background(), func(instanceTypeAggregatedList *compute.MachineTypeAggregatedList) error {
		if instanceTypeAggregatedList != nil {
			for _, scopedMachineTypes := range instanceTypeAggregatedList.Items {
				for _, machineType := range scopedMachineTypes.MachineTypes {
					if machineType == nil {
						continue
					}
					machineTypesList = append(machineTypesList, *machineType)
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return machineTypesList, nil
}

func (c *Client) GetInstanceTypesWithPrefix(prefix string, zone string) ([]string, error) {
	instanceTypes, err := c.GetAvailableInstanceTypes(zone)
	if err != nil {
		return nil, err
	}

	instanceTypesWithPrefix := strset.New()
	for _, instanceType := range instanceTypes {
		if strings.HasPrefix(instanceType, prefix) {
			instanceTypesWithPrefix.Add(instanceType)
		}
	}

	return instanceTypesWithPrefix.SliceSorted(), nil
}

func (c *Client) GetAvailableAcceleratorTypes(zone string) ([]string, error) {
	compClient, err := c.Compute()
	if err != nil {
		return nil, err
	}

	availableAcceleratorTypes := strset.New()

	err = compClient.AcceleratorTypes.List(c.ProjectID, zone).Pages(context.Background(), func(acceleratorTypeList *compute.AcceleratorTypeList) error {
		if acceleratorTypeList != nil {
			for _, acceleratorType := range acceleratorTypeList.Items {
				if acceleratorType == nil {
					continue
				}
				availableAcceleratorTypes.Add(acceleratorType.Name)
			}
		}
		return nil
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return availableAcceleratorTypes.SliceSorted(), nil
}

func (c *Client) GetAvailableAcceleratorTypesForAllZones() ([]string, error) {
	compClient, err := c.Compute()
	if err != nil {
		return nil, err
	}

	availableAcceleratorTypes := strset.New()

	err = compClient.AcceleratorTypes.AggregatedList(c.ProjectID).Pages(context.Background(), func(acceleratorTypeAggregatedList *compute.AcceleratorTypeAggregatedList) error {
		if acceleratorTypeAggregatedList != nil {
			for _, acceleratorAggregateType := range acceleratorTypeAggregatedList.Items {
				for _, accelerator := range acceleratorAggregateType.AcceleratorTypes {
					if accelerator == nil {
						continue
					}
					availableAcceleratorTypes.Add(accelerator.Name)
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return availableAcceleratorTypes.SliceSorted(), nil
}

func (c *Client) GetAvailableZonesForAccelerator(acceleratorType string) ([]string, error) {
	compClient, err := c.Compute()
	if err != nil {
		return nil, err
	}

	availableAcceleratorZones := strset.New()

	err = compClient.AcceleratorTypes.AggregatedList(c.ProjectID).Pages(context.Background(), func(acceleratorTypeAggregatedList *compute.AcceleratorTypeAggregatedList) error {
		if acceleratorTypeAggregatedList != nil {
			for _, acceleratorAggregateType := range acceleratorTypeAggregatedList.Items {
				for _, accelerator := range acceleratorAggregateType.AcceleratorTypes {
					if accelerator == nil {
						continue
					}
					if accelerator.Name == acceleratorType {
						splitZone := strings.Split(accelerator.Zone, "/")
						availableAcceleratorZones.Add(splitZone[len(splitZone)-1])
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return availableAcceleratorZones.SliceSorted(), nil
}

func (c *Client) ListDisks(zone string) ([]compute.Disk, error) {
	disksClient, err := c.Disks()
	if err != nil {
		return nil, err
	}

	var disks []compute.Disk

	err = disksClient.List(c.ProjectID, zone).Pages(context.Background(), func(list *compute.DiskList) error {
		for _, disk := range list.Items {
			if disk == nil {
				continue
			}
			disks = append(disks, *disk)
		}
		return nil
	})

	if err != nil {
		return nil, errors.WithStack(err)
	}

	return disks, nil
}
