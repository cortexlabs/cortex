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

package resources

import (
	"fmt"
	"strings"

	"github.com/cortexlabs/cortex/pkg/config"
	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/console"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/lib/slices"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/table"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"github.com/cortexlabs/cortex/pkg/workloads"
	istioclientnetworking "istio.io/client-go/pkg/apis/networking/v1beta1"
	kresource "k8s.io/apimachinery/pkg/api/resource"
)

func ValidateClusterAPIs(apis []userconfig.API) error {
	if len(apis) == 0 {
		return spec.ErrorNoAPIs()
	}

	virtualServices, err := config.K8s.ListVirtualServices(nil)
	if err != nil {
		return err
	}
	httpDeployedRealtimeAPIs := strset.New()
	for _, virtualService := range virtualServices {
		if virtualService.Labels["apiKind"] == userconfig.RealtimeAPIKind.String() {
			httpDeployedRealtimeAPIs.Add(virtualService.Labels["apiName"])
		}
	}

	realtimeAPIs := InclusiveFilterAPIsByKind(apis, userconfig.RealtimeAPIKind)

	for i := range apis {
		api := &apis[i]
		if api.Kind == userconfig.RealtimeAPIKind || api.Kind == userconfig.BatchAPIKind ||
			api.Kind == userconfig.TaskAPIKind || api.Kind == userconfig.AsyncAPIKind {

			if err := spec.ValidateAPI(api, config.AWS, config.K8s); err != nil {
				return errors.Wrap(err, api.Identify())
			}

			if err := validateEndpointCollisions(api, virtualServices); err != nil {
				return err
			}
		}

		if api.Kind == userconfig.TrafficSplitterKind {
			if err := spec.ValidateTrafficSplitter(api); err != nil {
				return errors.Wrap(err, api.Identify())
			}
			if err := checkIfAPIExists(api.APIs, realtimeAPIs, httpDeployedRealtimeAPIs); err != nil {
				return errors.Wrap(err, api.Identify())
			}
			if err := validateEndpointCollisions(api, virtualServices); err != nil {
				return errors.Wrap(err, api.Identify())
			}
		}
	}

	maxMemMap, err := operator.UpdateMemoryCapacityConfigMap()
	if err != nil {
		return err
	}

	for i := range apis {
		api := &apis[i]
		if api.Kind != userconfig.TrafficSplitterKind {
			if err := validateK8sCompute(api, maxMemMap); err != nil {
				return errors.Wrap(err, api.Identify())
			}
		}
	}

	dups := spec.FindDuplicateNames(apis)
	if len(dups) > 0 {
		return spec.ErrorDuplicateName(dups)
	}
	dups = findDuplicateEndpoints(apis)
	if len(dups) > 0 {
		return spec.ErrorDuplicateEndpointInOneDeploy(dups)
	}

	return nil
}

/*
CPU Reservations:

FluentBit 100
StatsDExporter 100
NodeExporter 110 (it has two containers)
KubeProxy 100
AWS cni 10
Reserved (150 + 150) see eks.yaml for details
*/
var _cortexCPUReserve = kresource.MustParse("720m")

/*
Memory Reservations:

FluentBit 150
StatsDExporter 100
NodeExporter 200 (it has two containers)
Reserved (300 + 300 + 200) see eks.yaml for details
*/
var _cortexMemReserve = kresource.MustParse("1250Mi")

var _nvidiaCPUReserve = kresource.MustParse("100m")
var _nvidiaMemReserve = kresource.MustParse("100Mi")

var _nvidiaDCGMExporterCPUReserve = kresource.MustParse("50m")
var _nvidiaDCGMExporterMemReserve = kresource.MustParse("50Mi")

var _inferentiaCPUReserve = kresource.MustParse("100m")
var _inferentiaMemReserve = kresource.MustParse("100Mi")

func validateK8sCompute(api *userconfig.API, maxMemMap map[string]kresource.Quantity) error {
	clusterNodeGroupNames := strset.New(config.ClusterConfig.GetNodeGroupNames()...)
	for _, ngName := range api.NodeGroups {
		if !clusterNodeGroupNames.Has(ngName) {
			return errors.Wrap(ErrorInvalidNodeGroupSelector(ngName, config.ClusterConfig.GetNodeGroupNames()), userconfig.NodeGroupsKey)
		}
	}

	compute := userconfig.GetPodComputeRequest(api)

	for _, ng := range config.ClusterConfig.NodeGroups {
		if api.NodeGroups != nil && !slices.HasString(api.NodeGroups, ng.Name) {
			continue
		}

		nodeCPU, nodeMem, nodeGPU, nodeInf := getNodeCapacity(ng.InstanceType, maxMemMap)

		if compute.CPU != nil && nodeCPU.Cmp(compute.CPU.Quantity) < 0 {
			continue
		} else if compute.Mem != nil && nodeMem.Cmp(compute.Mem.Quantity) < 0 {
			continue
		} else if compute.GPU > nodeGPU {
			continue
		} else if compute.Inf > nodeInf {
			continue
		}

		// we found a node group that has capacity
		return nil
	}

	// no nodegroups have capacity
	errMsg := "no instance types in your cluster are large enough to satisfy the requested resources for your pod\n\n"
	errMsg += console.Bold("requested pod resources\n")
	errMsg += podResourceRequestsTable(api, compute)
	errMsg += "\n" + s.TrimTrailingNewLines(nodeGroupResourcesTable(api, compute, maxMemMap))
	return ErrorNoAvailableNodeComputeLimit(errMsg)
}

func podResourceRequestsTable(api *userconfig.API, compute userconfig.Compute) string {
	sidecarCPUNote := ""
	sidecarMemNote := ""
	if api.Kind == userconfig.RealtimeAPIKind {
		sidecarCPUNote = fmt.Sprintf(" (including %s for the %s sidecar container)", consts.CortexProxyCPU.String(), workloads.ProxyContainerName)
		sidecarMemNote = fmt.Sprintf(" (including %s for the %s sidecar container)", k8s.ToMiCeilStr(consts.CortexProxyMem), workloads.ProxyContainerName)
	} else if api.Kind == userconfig.AsyncAPIKind || api.Kind == userconfig.BatchAPIKind {
		sidecarCPUNote = fmt.Sprintf(" (including %s for the %s sidecar container)", consts.CortexDequeuerCPU.String(), workloads.DequeuerContainerName)
		sidecarMemNote = fmt.Sprintf(" (including %s for the %s sidecar container)", k8s.ToMiCeilStr(consts.CortexDequeuerMem), workloads.DequeuerContainerName)
	}

	var items table.KeyValuePairs
	if compute.CPU != nil {
		items.Add("CPU", compute.CPU.String()+sidecarCPUNote)
	}
	if compute.Mem != nil {
		items.Add("memory", compute.Mem.ToMiCeilStr()+sidecarMemNote)
	}
	if compute.GPU > 0 {
		items.Add("GPU", compute.GPU)
	}
	if compute.Inf > 0 {
		items.Add("Inf", compute.Inf)
	}

	return items.String()
}

func nodeGroupResourcesTable(api *userconfig.API, compute userconfig.Compute, maxMemMap map[string]kresource.Quantity) string {
	var skippedNodeGroups []string
	var nodeGroupResourceRows [][]interface{}

	showGPU := false
	showInf := false
	if compute.GPU > 0 {
		showGPU = true
	}
	if compute.Inf > 0 {
		showInf = true
	}

	for _, ng := range config.ClusterConfig.NodeGroups {
		nodeCPU, nodeMem, nodeGPU, nodeInf := getNodeCapacity(ng.InstanceType, maxMemMap)
		if nodeGPU > 0 {
			showGPU = true
		}
		if nodeInf > 0 {
			showInf = true
		}

		if api.NodeGroups != nil && !slices.HasString(api.NodeGroups, ng.Name) {
			skippedNodeGroups = append(skippedNodeGroups, ng.Name)
		} else {
			nodeGroupResourceRows = append(nodeGroupResourceRows, []interface{}{ng.Name, ng.InstanceType, nodeCPU, k8s.ToMiFloorStr(nodeMem), nodeGPU, nodeInf})
		}
	}

	nodeGroupResourceRowsTable := table.Table{
		Headers: []table.Header{
			{Title: "node group"},
			{Title: "instance type"},
			{Title: "CPU"},
			{Title: "memory"},
			{Title: "GPU", Hidden: !showGPU},
			{Title: "Inf", Hidden: !showInf},
		},
		Rows: nodeGroupResourceRows,
	}

	out := nodeGroupResourceRowsTable.MustFormat()
	if len(skippedNodeGroups) > 0 {
		out += fmt.Sprintf("\nthe following %s skipped (based on the api configuration's %s field): %s", s.PluralCustom("node group was", "node groups were", len(skippedNodeGroups)), userconfig.NodeGroupsKey, strings.Join(skippedNodeGroups, ", "))
	}

	return out
}

func getNodeCapacity(instanceType string, maxMemMap map[string]kresource.Quantity) (kresource.Quantity, kresource.Quantity, int64, int64) {
	instanceMetadata := aws.InstanceMetadatas[config.ClusterConfig.Region][instanceType]

	cpu := instanceMetadata.CPU.DeepCopy()
	cpu.Sub(_cortexCPUReserve)

	mem := maxMemMap[instanceType].DeepCopy()
	mem.Sub(_cortexMemReserve)

	gpu := instanceMetadata.GPU
	if gpu > 0 {
		// Reserve resources for nvidia device plugin daemonset
		cpu.Sub(_nvidiaCPUReserve)
		mem.Sub(_nvidiaMemReserve)
		// Reserve resources for nvidia dcgm prometheus exporter
		cpu.Sub(_nvidiaDCGMExporterCPUReserve)
		mem.Sub(_nvidiaDCGMExporterMemReserve)
	}

	inf := instanceMetadata.Inf
	if inf > 0 {
		// Reserve resources for inferentia device plugin daemonset
		cpu.Sub(_inferentiaCPUReserve)
		mem.Sub(_inferentiaMemReserve)
	}

	return cpu, mem, gpu, inf
}

func validateEndpointCollisions(api *userconfig.API, virtualServices []istioclientnetworking.VirtualService) error {
	for i := range virtualServices {
		virtualService := virtualServices[i]
		gateways := k8s.ExtractVirtualServiceGateways(&virtualService)
		if !gateways.Has("apis-gateway") {
			continue
		}

		endpoints := k8s.ExtractVirtualServiceEndpoints(&virtualService)
		for endpoint := range endpoints {
			if s.EnsureSuffix(endpoint, "/") == s.EnsureSuffix(*api.Networking.Endpoint, "/") && virtualService.Labels["apiName"] != api.Name {
				return errors.Wrap(spec.ErrorDuplicateEndpoint(virtualService.Labels["apiName"]), userconfig.NetworkingKey, userconfig.EndpointKey, endpoint)
			}
		}
	}

	return nil
}

func findDuplicateEndpoints(apis []userconfig.API) []userconfig.API {
	endpoints := make(map[string][]userconfig.API)

	for _, api := range apis {
		endpoints[*api.Networking.Endpoint] = append(endpoints[*api.Networking.Endpoint], api)
	}

	for endpoint := range endpoints {
		if len(endpoints[endpoint]) > 1 {
			return endpoints[endpoint]
		}
	}

	return nil
}

// InclusiveFilterAPIsByKind includes only provided Kinds
func InclusiveFilterAPIsByKind(apis []userconfig.API, kindsToInclude ...userconfig.Kind) []userconfig.API {
	kindsToIncludeSet := strset.New()
	for _, kind := range kindsToInclude {
		kindsToIncludeSet.Add(kind.String())
	}
	fileredAPIs := []userconfig.API{}
	for _, api := range apis {
		if kindsToIncludeSet.Has(api.Kind.String()) {
			fileredAPIs = append(fileredAPIs, api)
		}
	}
	return fileredAPIs
}

func ExclusiveFilterAPIsByKind(apis []userconfig.API, kindsToExclude ...userconfig.Kind) []userconfig.API {
	kindsToExcludeSet := strset.New()
	for _, kind := range kindsToExclude {
		kindsToExcludeSet.Add(kind.String())
	}
	fileredAPIs := []userconfig.API{}
	for _, api := range apis {
		if !kindsToExcludeSet.Has(api.Kind.String()) {
			fileredAPIs = append(fileredAPIs, api)
		}
	}
	return fileredAPIs
}

// checkIfAPIExists checks if referenced apis in trafficsplitter are either defined in yaml or already deployed.
func checkIfAPIExists(trafficSplitterAPIs []*userconfig.TrafficSplit, apis []userconfig.API, httpDeployedRealtimeAPIs strset.Set) error {
	var missingAPIs []string
	// check if apis named in trafficsplitter are either defined in same yaml or already deployed
	for _, trafficSplitAPI := range trafficSplitterAPIs {
		// check if already deployed
		deployed := httpDeployedRealtimeAPIs.Has(trafficSplitAPI.Name)

		// check defined apis
		for _, definedAPI := range apis {
			if trafficSplitAPI.Name == definedAPI.Name {
				deployed = true
			}
		}
		if !deployed {
			missingAPIs = append(missingAPIs, trafficSplitAPI.Name)
		}
	}
	if len(missingAPIs) != 0 {
		return ErrorAPIsNotDeployed(missingAPIs)
	}
	return nil

}
