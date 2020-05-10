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

package endpoints

import (
	"net/http"
	"os"
	"strings"

	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	kcore "k8s.io/api/core/v1"
)

func Info(w http.ResponseWriter, r *http.Request) {
	nodeInfos, err := getNodeInfos()
	if err != nil {
		respondError(w, r, err)
		return
	}

	response := schema.InfoResponse{
		MaskedAWSAccessKeyID: s.MaskString(os.Getenv("AWS_ACCESS_KEY_ID"), 4),
		ClusterConfig:        *config.Cluster,
		NodeInfos:            nodeInfos,
	}
	respond(w, response)
}

func getNodeInfos() ([]schema.NodeInfo, error) {
	pods, err := config.K8sAllNamspaces.ListPods(nil)
	if err != nil {
		return nil, err
	}

	nodes, err := config.K8sAllNamspaces.ListNodesByLabel("workload", "true")
	if err != nil {
		return nil, err
	}

	nodeInfoMap := make(map[string]*schema.NodeInfo, len(nodes)) // node name -> info
	spotPriceCache := make(map[string]float64)                   // instance type -> spot price

	for _, node := range nodes {
		instanceType := node.Labels["beta.kubernetes.io/instance-type"]
		isSpot := strings.Contains(strings.ToLower(node.Labels["lifecycle"]), "spot")

		price := aws.InstanceMetadatas[*config.Cluster.Region][instanceType].Price
		if isSpot {
			if spotPrice, ok := spotPriceCache[instanceType]; ok {
				price = spotPrice
			} else {
				spotPrice, err := config.AWS.SpotInstancePrice(*config.Cluster.Region, instanceType)
				if err == nil && spotPrice != 0 {
					price = spotPrice
					spotPriceCache[instanceType] = spotPrice
				} else {
					spotPriceCache[instanceType] = price // the request failed, so no need to try again in the future
				}
			}
		}

		nodeInfoMap[node.Name] = &schema.NodeInfo{
			InstanceType:     instanceType,
			IsSpot:           isSpot,
			Price:            price,
			NumReplicas:      0,                             // will be added to below
			ComputeCapacity:  nodeComputeAllocatable(&node), // will be subtracted from below
			ComputeAvailable: nodeComputeAllocatable(&node), // will be subtracted from below
		}
	}

	for _, pod := range pods {
		node, ok := nodeInfoMap[pod.Spec.NodeName]
		if !ok {
			continue
		}

		_, isAPIPod := pod.Labels["apiName"]

		if isAPIPod {
			node.NumReplicas++
		}

		cpu, mem, gpu := k8s.TotalPodCompute(&pod.Spec)

		node.ComputeAvailable.CPU.SubQty(cpu)
		node.ComputeAvailable.Mem.SubQty(mem)
		node.ComputeAvailable.GPU -= gpu

		if !isAPIPod {
			node.ComputeCapacity.CPU.SubQty(cpu)
			node.ComputeCapacity.Mem.SubQty(mem)
			node.ComputeCapacity.GPU -= gpu
		}
	}

	nodeInfos := make([]schema.NodeInfo, 0, len(nodeInfoMap))
	for _, nodeInfo := range nodeInfoMap {
		nodeInfos = append(nodeInfos, *nodeInfo)
	}

	return nodeInfos, nil
}

func nodeComputeAllocatable(node *kcore.Node) userconfig.Compute {
	gpuQty := node.Status.Allocatable["nvidia.com/gpu"]

	return userconfig.Compute{
		CPU: k8s.WrapQuantity(*node.Status.Allocatable.Cpu()),
		Mem: k8s.WrapQuantity(*node.Status.Allocatable.Memory()),
		GPU: (&gpuQty).Value(),
	}
}
