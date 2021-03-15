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
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/slices"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	kresource "k8s.io/apimachinery/pkg/api/resource"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
)

const _memConfigMapName = "cortex-instance-memory"
const _configKeyPrefix = "memory-capacity-"

func getMemoryCapacityFromNodes(primaryInstances []string) (map[string]*kresource.Quantity, error) {
	opts := kmeta.ListOptions{
		LabelSelector: klabels.SelectorFromSet(map[string]string{
			"workload": "true",
		}).String(),
	}
	nodes, err := config.K8s.ListNodes(&opts)
	if err != nil {
		return nil, err
	}

	minMemMap := map[string]*kresource.Quantity{}
	for _, primaryInstance := range primaryInstances {
		minMemMap[primaryInstance] = nil
	}

	for _, node := range nodes {
		isPrimaryInstance := false
		var primaryInstanceType string
		for k, v := range node.Labels {
			if k == "beta.kubernetes.io/instance-type" && slices.HasString(primaryInstances, v) {
				isPrimaryInstance = true
				primaryInstanceType = v
				break
			}
		}
		if !isPrimaryInstance {
			continue
		}

		curMem := node.Status.Capacity.Memory()

		if curMem == nil || curMem.IsZero() {
			continue
		}

		if minMemMap[primaryInstanceType] == nil || minMemMap[primaryInstanceType].Cmp(*curMem) > 0 {
			minMemMap[primaryInstanceType] = curMem
		}
	}

	return minMemMap, nil
}

func getMemoryCapacityFromConfigMap() (map[string]*kresource.Quantity, error) {
	configMapData, err := config.K8s.GetConfigMapData(_memConfigMapName)
	if err != nil {
		return nil, err
	}

	if len(configMapData) == 0 {
		return nil, nil
	}

	memoryCapacitiesMap := map[string]*kresource.Quantity{}
	for k := range configMapData {
		memoryUserStr := configMapData[k]
		mem, err := kresource.ParseQuantity(memoryUserStr)
		if err != nil {
			return nil, err
		}
		instanceType := k[len(_configKeyPrefix):]
		if mem.IsZero() {
			memoryCapacitiesMap[instanceType] = nil
		} else {
			memoryCapacitiesMap[instanceType] = &mem
		}
	}

	return memoryCapacitiesMap, nil
}

func UpdateMemoryCapacityConfigMap() (map[string]kresource.Quantity, error) {
	if !config.IsManaged() {
		return nil, nil
	}

	instancesMetadata := config.AWSInstancesMetadata()
	if len(instancesMetadata) == 0 {
		return nil, errors.ErrorUnexpected("unable to find instances metadata; likely because this is not a cortex managed cluster")
	}
	primaryInstances := []string{}

	minMemMap := map[string]kresource.Quantity{}
	for _, instanceMetadata := range instancesMetadata {
		minMemMap[instanceMetadata.Type] = instanceMetadata.Memory
		primaryInstances = append(primaryInstances, instanceMetadata.Type)
	}

	nodeMemCapacityMap, err := getMemoryCapacityFromNodes(primaryInstances)
	if err != nil {
		return nil, err
	}

	previousMinMemMap, err := getMemoryCapacityFromConfigMap()
	if err != nil {
		return nil, err
	}

	configMapData := map[string]string{}
	for _, primaryInstance := range primaryInstances {
		minMem := minMemMap[primaryInstance]

		if nodeMemCapacityMap[primaryInstance] != nil && minMem.Cmp(*nodeMemCapacityMap[primaryInstance]) > 0 {
			minMem = *nodeMemCapacityMap[primaryInstance]
		}

		if previousMinMemMap[primaryInstance] != nil && minMem.Cmp(*previousMinMemMap[primaryInstance]) > 0 {
			minMem = *previousMinMemMap[primaryInstance]
		}

		if previousMinMemMap[primaryInstance] == nil || minMem.Cmp(*previousMinMemMap[primaryInstance]) < 0 {
			configMapData[_configKeyPrefix+primaryInstance] = minMem.String()
		} else {
			configMapData[_configKeyPrefix+primaryInstance] = previousMinMemMap[primaryInstance].String()
		}

		minMemMap[primaryInstance] = minMem
	}

	configMap := k8s.ConfigMap(&k8s.ConfigMapSpec{
		Name: _memConfigMapName,
		Data: configMapData,
	})

	_, err = config.K8s.ApplyConfigMap(configMap)
	if err != nil {
		return nil, err
	}

	return minMemMap, nil
}
