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

package endpoints

import (
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types"
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	kcore "k8s.io/api/core/v1"
)

func Info(w http.ResponseWriter, r *http.Request) {
	if config.Provider == types.AWSProviderType {
		nodeInfos, numPendingReplicas, err := getNodeInfos()
		if err != nil {
			respondError(w, r, err)
			return
		}

		fullClusterConfig := clusterconfig.InternalConfig{
			Config: clusterconfig.Config{
				CoreConfig: *config.CoreConfig,
			},
			OperatorMetadata: *config.OperatorMetadata,
		}

		if config.IsManaged() {
			fullClusterConfig.Config.ManagedConfig = *config.ManagedConfigOrNil()
			fullClusterConfig.InstancesMetadata = config.AWSInstancesMetadata()
		}

		response := schema.InfoResponse{
			MaskedAWSAccessKeyID: s.MaskString(os.Getenv("AWS_ACCESS_KEY_ID"), 4),
			ClusterConfig:        fullClusterConfig,
			NodeInfos:            nodeInfos,
			NumPendingReplicas:   numPendingReplicas,
		}
		respond(w, response)
	} else {
		fullClusterConfig := clusterconfig.InternalGCPConfig{
			GCPConfig: clusterconfig.GCPConfig{
				GCPCoreConfig: *config.GCPCoreConfig,
			},
			OperatorMetadata: *config.OperatorMetadata,
		}

		if config.IsManaged() {
			fullClusterConfig.GCPConfig.GCPManagedConfig = *config.GCPManagedConfigOrNil()
		}

		response := schema.InfoGCPResponse{
			ClusterConfig: fullClusterConfig,
		}
		respond(w, response)
	}
}

func getNodeInfos() ([]schema.NodeInfo, int, error) {
	pods, err := config.K8sAllNamspaces.ListPods(nil)
	if err != nil {
		return nil, 0, err
	}

	nodes, err := config.K8sAllNamspaces.ListNodesByLabel("workload", "true")
	if err != nil {
		return nil, 0, err
	}

	nodeInfoMap := make(map[string]*schema.NodeInfo, len(nodes)) // node name -> info
	spotPriceCache := make(map[string]float64)                   // instance type -> spot price

	for i := range nodes {
		node := nodes[i]

		instanceType := node.Labels["beta.kubernetes.io/instance-type"]
		nodeGroupName := node.Labels["alpha.eksctl.io/nodegroup-name"]
		isSpot := strings.Contains(strings.ToLower(node.Labels["lifecycle"]), "spot")

		price := aws.InstanceMetadatas[config.CoreConfig.Region][instanceType].Price
		if isSpot {
			if spotPrice, ok := spotPriceCache[instanceType]; ok {
				price = spotPrice
			} else {
				spotPrice, err := config.AWS.SpotInstancePrice(instanceType)
				if err == nil && spotPrice != 0 {
					price = spotPrice
					spotPriceCache[instanceType] = spotPrice
				} else {
					spotPriceCache[instanceType] = price // the request failed, so no need to try again in the future
				}
			}
		}

		nodeInfoMap[node.Name] = &schema.NodeInfo{
			Name:                 node.Name,
			NodeGroupName:        nodeGroupName,
			InstanceType:         instanceType,
			IsSpot:               isSpot,
			Price:                price,
			NumReplicas:          0,                             // will be added to below
			ComputeUserCapacity:  nodeComputeAllocatable(&node), // will be subtracted from below
			ComputeAvailable:     nodeComputeAllocatable(&node), // will be subtracted from below
			ComputeUserRequested: userconfig.ZeroCompute(),      // will be added to below
		}
	}

	var numPendingReplicas int

	for i := range pods {
		pod := pods[i]

		_, isAPIPod := pod.Labels["apiName"]

		if pod.Spec.NodeName == "" && isAPIPod {
			numPendingReplicas++
			continue
		}

		node, ok := nodeInfoMap[pod.Spec.NodeName]
		if !ok {
			continue
		}

		if isAPIPod {
			node.NumReplicas++
		}

		cpu, mem, gpu, inf := k8s.TotalPodCompute(&pod.Spec)

		node.ComputeAvailable.CPU.SubQty(cpu)
		node.ComputeAvailable.Mem.SubQty(mem)
		node.ComputeAvailable.GPU -= gpu
		node.ComputeAvailable.Inf -= inf

		if isAPIPod {
			node.ComputeUserRequested.CPU.AddQty(cpu)
			node.ComputeUserRequested.Mem.AddQty(mem)
			node.ComputeUserRequested.GPU += gpu
			node.ComputeUserRequested.Inf += inf
		} else {
			node.ComputeUserCapacity.CPU.SubQty(cpu)
			node.ComputeUserCapacity.Mem.SubQty(mem)
			node.ComputeUserCapacity.GPU -= gpu
			node.ComputeUserCapacity.Inf -= inf
		}
	}

	nodeNames := make([]string, 0, len(nodeInfoMap))
	for nodeName := range nodeInfoMap {
		nodeNames = append(nodeNames, nodeName)
	}

	sort.Strings(nodeNames)

	nodeInfos := make([]schema.NodeInfo, len(nodeNames))
	for i, nodeName := range nodeNames {
		nodeInfos[i] = *nodeInfoMap[nodeName]
	}

	return nodeInfos, numPendingReplicas, nil
}

func nodeComputeAllocatable(node *kcore.Node) userconfig.Compute {
	gpuQty := node.Status.Allocatable["nvidia.com/gpu"]
	infQty := node.Status.Allocatable["aws.amazon.com/neuron"]

	return userconfig.Compute{
		CPU: k8s.WrapQuantity(*node.Status.Allocatable.Cpu()),
		Mem: k8s.WrapQuantity(*node.Status.Allocatable.Memory()),
		GPU: gpuQty.Value(),
		Inf: infQty.Value(),
	}
}
