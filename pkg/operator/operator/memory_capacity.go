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

package operator

import (
	"math"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/types"
	kresource "k8s.io/apimachinery/pkg/api/resource"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
)

const _memConfigMapName = "cortex-instance-memory"
const _memConfigMapKey = "capacity"

func getMemoryCapacityFromNodes() (*kresource.Quantity, error) {
	opts := kmeta.ListOptions{
		LabelSelector: klabels.SelectorFromSet(map[string]string{
			"workload": "true",
		}).String(),
	}
	nodes, err := config.K8s.ListNodes(&opts)
	if err != nil {
		return nil, err
	}

	var minMem *kresource.Quantity
	for _, node := range nodes {
		curMem := node.Status.Capacity.Memory()

		if curMem == nil || curMem.IsZero() {
			continue
		}

		if minMem == nil || minMem.Cmp(*curMem) < 0 {
			minMem = curMem
		}
	}

	return minMem, nil
}

func getMemoryCapacityFromConfigMap() (*kresource.Quantity, error) {
	configMapData, err := config.K8s.GetConfigMapData(_memConfigMapName)
	if err != nil {
		return nil, err
	}

	if len(configMapData) == 0 {
		return nil, nil
	}

	memoryUserStr := configMapData[_memConfigMapKey]
	mem, err := kresource.ParseQuantity(memoryUserStr)
	if err != nil {
		return nil, err
	}
	if mem.IsZero() {
		return nil, nil
	}
	return &mem, nil
}

func UpdateMemoryCapacityConfigMap() (kresource.Quantity, error) {
	if !config.IsManaged() {
		return kresource.Quantity{}, nil
	}

	minMem := *kresource.NewQuantity(math.MaxInt64, kresource.DecimalSI)

	if config.Provider == types.AWSProviderType {
		instanceMetadata := config.AWSInstanceMetadataOrNil()
		if instanceMetadata == nil {
			return kresource.Quantity{}, errors.ErrorUnexpected("unable to find instance metadata; likely because this is not a cortex managed cluster")
		}
		minMem = instanceMetadata.Memory
	}

	nodeMemCapacity, err := getMemoryCapacityFromNodes()
	if err != nil {
		return kresource.Quantity{}, err
	}

	previousMinMem, err := getMemoryCapacityFromConfigMap()
	if err != nil {
		return kresource.Quantity{}, err
	}

	if nodeMemCapacity != nil && minMem.Cmp(*nodeMemCapacity) > 0 {
		minMem = *nodeMemCapacity
	}

	if previousMinMem != nil && minMem.Cmp(*previousMinMem) > 0 {
		minMem = *previousMinMem
	}

	if previousMinMem == nil || minMem.Cmp(*previousMinMem) < 0 {
		configMap := k8s.ConfigMap(&k8s.ConfigMapSpec{
			Name: _memConfigMapName,
			Data: map[string]string{
				_memConfigMapKey: minMem.String(),
			},
		})

		_, err := config.K8s.ApplyConfigMap(configMap)
		if err != nil {
			return kresource.Quantity{}, err
		}
	}

	return minMem, nil
}
