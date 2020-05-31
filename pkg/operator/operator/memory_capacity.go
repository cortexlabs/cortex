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

package operator

import (
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	kresource "k8s.io/apimachinery/pkg/api/resource"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
)

const _memConfigMapName = "cortex-instance-memory"
const _memConfigMapKey = "capacity"

/*
CPU Reservations:

FluentD 200
StatsD 100
KubeProxy 100
AWS cni 10
Reserved (150 + 150) see eks.yaml for details
*/
var _cortexCPUReserve = kresource.MustParse("710m")

/*
Memory Reservations:

FluentD 200
StatsD 100
Reserved (300 + 300 + 200) see eks.yaml for details
*/
var _cortexMemReserve = kresource.MustParse("1100Mi")

var _nvidiaCPUReserve = kresource.MustParse("100m")
var _nvidiaMemReserve = kresource.MustParse("100Mi")

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

		if curMem != nil && minMem == nil {
			minMem = curMem
		}

		if curMem != nil && minMem.Cmp(*curMem) < 0 {
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
	return &mem, nil
}

func updateMemoryCapacityConfigMap() (*kresource.Quantity, error) {
	memFromConfig := config.Cluster.InstanceMetadata.Memory
	memFromNodes, err := getMemoryCapacityFromNodes()
	if err != nil {
		return nil, err
	}

	memFromConfigMap, err := getMemoryCapacityFromConfigMap()
	if err != nil {
		return nil, err
	}

	minMem := k8s.QuantityPtr(memFromConfig.DeepCopy())

	if memFromNodes != nil && minMem.Cmp(*memFromNodes) > 0 {
		minMem = memFromNodes
	}

	if memFromConfigMap != nil && minMem.Cmp(*memFromConfigMap) > 0 {
		minMem = memFromConfigMap
	}

	if memFromConfigMap == nil || minMem.Cmp(*memFromConfigMap) != 0 {
		configMap := k8s.ConfigMap(&k8s.ConfigMapSpec{
			Name: _memConfigMapName,
			Data: map[string]string{
				_memConfigMapKey: minMem.String(),
			},
		})

		_, err := config.K8s.ApplyConfigMap(configMap)
		if err != nil {
			return nil, err
		}
	}

	return minMem, nil
}
