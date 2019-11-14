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

package workloads

import (
	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	kresource "k8s.io/apimachinery/pkg/api/resource"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const key = "capacity"
const configMemoryMapName = "cortex-instance-memory"

func GetMemoryCapacityFromNodes() (*kresource.Quantity, error) {
	opts := kmeta.ListOptions{
		LabelSelector: k8s.LabelSelector(map[string]string{
			"workload": "true",
		}),
	}
	nodes, err := config.Kubernetes.ListNodes(&opts)
	if err != nil {
		return nil, err
	}

	var minMem *kresource.Quantity
	for _, node := range nodes {
		curMem := node.Status.Capacity.Memory()

		if curMem != nil && minMem == nil {
			minMem = curMem
		}

		if curMem != nil && minMem.Cmp(*curMem) < 0 {
			minMem = curMem
		}
	}

	return minMem, nil
}

func GetMemoryCapacityFromConfigMap() (*kresource.Quantity, error) {
	configMap, err := config.Kubernetes.GetConfigMap(configMemoryMapName)
	if err != nil {
		return nil, err
	}

	if configMap == nil {
		return nil, nil
	}

	memoryUserStr := configMap.Data[key]
	mem, err := kresource.ParseQuantity(memoryUserStr)
	if err != nil {
		return nil, err
	}
	return &mem, nil
}

func UpdateMemoryCapacityConfigMap() (*kresource.Quantity, error) {
	memFromConfig := config.Cluster.InstanceMem
	memFromNodes, err := GetMemoryCapacityFromNodes()
	if err != nil {
		return nil, err
	}

	memFromConfigMap, err := GetMemoryCapacityFromConfigMap()
	if err != nil {
		return nil, err
	}

	minMem := memFromConfig.Copy()

	if memFromNodes != nil && minMem.Cmp(*memFromNodes) > 0 {
		minMem = memFromNodes
	}

	if memFromConfigMap != nil && minMem.Cmp(*memFromConfigMap) > 0 {
		minMem = memFromConfigMap
	}

	if minMem != memFromConfigMap {
		configMap := k8s.ConfigMap(&k8s.ConfigMapSpec{
			Name:      configMemoryMapName,
			Namespace: consts.K8sNamespace,
			Data: map[string]string{
				key: minMem.String(),
			},
		})

		_, err := config.Kubernetes.ApplyConfigMap(configMap)
		if err != nil {
			return nil, err
		}
	}
	return minMem, nil
}
