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

package gcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
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
		if IsErrCode(err, 404, pointer.String(errorMessage)) {
			return false, nil
		}
		return false, errors.WithStack(err)
	}

	return true, nil
}

func (c *Client) IsZoneValid() (bool, error) {
	compClient, err := c.Compute()
	if err != nil {
		return false, err
	}

	_, err = compClient.Zones.Get(c.ProjectID, c.Zone).Do()
	if err != nil {
		errorMessage := fmt.Sprintf("The resource 'projects/%s/zones/%s' was not found", c.ProjectID, c.Zone)
		if IsErrCode(err, 404, pointer.String(errorMessage)) {
			return false, nil
		}
		return false, errors.WithStack(err)
	}

	return true, nil
}

func (c *Client) IsInstanceTypeAvailable(instanceType string) (bool, error) {
	compClient, err := c.Compute()
	if err != nil {
		return false, err
	}

	_, err = compClient.MachineTypes.Get(c.ProjectID, c.Zone, instanceType).Do()
	if err != nil {
		errorMessage := fmt.Sprintf("The resource 'projects/%s/zones/%s/machineTypes/%s' was not found", c.ProjectID, c.Zone, instanceType)
		if IsErrCode(err, 404, pointer.String(errorMessage)) {
			return false, nil
		}
		return false, errors.WithStack(err)
	}

	return true, nil
}

func (c *Client) IsAcceleratorTypeAvailable(acceleratorType string) (bool, error) {
	compClient, err := c.Compute()
	if err != nil {
		return false, err
	}

	_, err = compClient.AcceleratorTypes.Get(c.ProjectID, c.Zone, acceleratorType).Do()
	if err != nil {
		resource := fmt.Sprintf("projects/%s/zones/%s/acceleratorTypes/%s", c.ProjectID, c.Zone, acceleratorType)
		if IsErrCode(err, 404, pointer.String(fmt.Sprintf("The resource '%s' was not found", resource))) {
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

func (c *Client) GetAvailableInstanceTypes() ([]string, error) {
	compClient, err := c.Compute()
	if err != nil {
		return nil, err
	}

	availableInstanceTypes := strset.New()

	err = compClient.MachineTypes.List(c.ProjectID, c.Zone).Pages(context.Background(), func(instanceTypeList *compute.MachineTypeList) error {
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

func (c *Client) GetInstanceTypesMetadata() ([]compute.MachineType, error) {
	compClient, err := c.Compute()
	if err != nil {
		return nil, err
	}

	machineTypesList := make([]compute.MachineType, 0)
	err = compClient.MachineTypes.List(c.ProjectID, c.Zone).Pages(context.Background(), func(instanceTypeList *compute.MachineTypeList) error {
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

func (c *Client) GetInstanceTypesWithPrefix(prefix string) ([]string, error) {
	instanceTypes, err := c.GetAvailableInstanceTypes()
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

func (c *Client) GetAvailableAcceleratorTypes() ([]string, error) {
	compClient, err := c.Compute()
	if err != nil {
		return nil, err
	}

	availableAcceleratorTypes := strset.New()

	err = compClient.AcceleratorTypes.List(c.ProjectID, c.Zone).Pages(context.Background(), func(acceleratorTypeList *compute.AcceleratorTypeList) error {
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
