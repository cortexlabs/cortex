/*
Copyright 2022 Cortex Labs, Inc.

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

package k8s

import (
	"context"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	kcore "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
)

var _nodeTypeMeta = kmeta.TypeMeta{
	APIVersion: "v1",
	Kind:       "Node",
}

func (c *Client) ListNodes(opts *kmeta.ListOptions) ([]kcore.Node, error) {
	if opts == nil {
		opts = &kmeta.ListOptions{}
	}
	nodeList, err := c.nodeClient.List(context.Background(), *opts)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	for i := range nodeList.Items {
		nodeList.Items[i].TypeMeta = _nodeTypeMeta
	}
	return nodeList.Items, nil
}

func (c *Client) ListNodesByLabels(labels map[string]string) ([]kcore.Node, error) {
	opts := &kmeta.ListOptions{
		LabelSelector: klabels.SelectorFromSet(labels).String(),
	}
	return c.ListNodes(opts)
}

func (c *Client) ListNodesByLabel(labelKey string, labelValue string) ([]kcore.Node, error) {
	return c.ListNodesByLabels(map[string]string{labelKey: labelValue})
}

func (c *Client) ListNodesWithLabelKeys(labelKeys ...string) ([]kcore.Node, error) {
	opts := &kmeta.ListOptions{
		LabelSelector: LabelExistsSelector(labelKeys...),
	}
	return c.ListNodes(opts)
}

func HowManyPodsFitOnNode(podSpec kcore.PodSpec, node kcore.Node, cpuReserved resource.Quantity, memoryReserved resource.Quantity) int64 {
	cpuQty := node.Status.Allocatable[v1.ResourceCPU]
	memoryQty := node.Status.Allocatable[v1.ResourceMemory]
	gpuQty := node.Status.Allocatable["nvidia.com/gpu"]
	infQty := node.Status.Allocatable["aws.amazon.com/neuron"]
	podsQty := node.Status.Allocatable[v1.ResourcePods]

	cpuQty.Sub(cpuReserved)
	memoryReserved.Sub(memoryReserved)

	cpuInt64 := cpuQty.MilliValue()
	memoryInt64 := memoryQty.MilliValue()
	gpuInt64 := gpuQty.Value()
	infInt64 := infQty.Value()
	maxPodsInt64 := podsQty.Value()

	cpuPodQty, memoryPodQty, podGPUInt64, podInfInt64 := TotalPodCompute(&podSpec)
	podCPUInt64 := cpuPodQty.MilliValue()
	podMemoryInt64 := memoryPodQty.MilliValue()

	if podCPUInt64 > 0 && float64(cpuInt64)/float64(podCPUInt64) < float64(maxPodsInt64) {
		maxPodsInt64 = int64(float64(cpuInt64) / float64(podCPUInt64))
	}
	if podMemoryInt64 > 0 && float64(memoryInt64)/float64(podMemoryInt64) < float64(maxPodsInt64) {
		maxPodsInt64 = int64(float64(memoryInt64) / float64(podMemoryInt64))
	}
	if podGPUInt64 > 0 && float64(gpuInt64)/float64(podGPUInt64) < float64(maxPodsInt64) {
		maxPodsInt64 = int64(float64(gpuInt64) / float64(podGPUInt64))
	}
	if podInfInt64 > 0 && float64(infInt64)/float64(podInfInt64) < float64(maxPodsInt64) {
		maxPodsInt64 = int64(float64(infInt64) / float64(podInfInt64))
	}

	return maxPodsInt64
}
