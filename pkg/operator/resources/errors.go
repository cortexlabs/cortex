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

package resources

import (
	"fmt"
	"strings"

	"github.com/cortexlabs/cortex/pkg/config"
	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/console"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/slices"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/table"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"github.com/cortexlabs/cortex/pkg/workloads"
	kresource "k8s.io/apimachinery/pkg/api/resource"
)

const (
	ErrOperationIsOnlySupportedForKind  = "resources.operation_is_only_supported_for_kind"
	ErrAPINotDeployed                   = "resources.api_not_deployed"
	ErrAPIIDNotFound                    = "resources.api_id_not_found"
	ErrCannotChangeTypeOfDeployedAPI    = "resources.cannot_change_kind_of_deployed_api"
	ErrNoAvailableNodeComputeLimit      = "resources.no_available_node_compute_limit"
	ErrJobIDRequired                    = "resources.job_id_required"
	ErrRealtimeAPIUsedByTrafficSplitter = "resources.realtime_api_used_by_traffic_splitter"
	ErrAPIsNotDeployed                  = "resources.apis_not_deployed"
	ErrInvalidNodeGroupSelector         = "resources.invalid_node_group_selector"
	ErrNoNodeGroups                     = "resources.no_node_groups"
)

func ErrorOperationIsOnlySupportedForKind(resource operator.DeployedResource, supportedKind userconfig.Kind, supportedKinds ...userconfig.Kind) error {
	supportedKindsSlice := append(make([]string, 0, 1+len(supportedKinds)), supportedKind.String())
	for _, kind := range supportedKinds {
		supportedKindsSlice = append(supportedKindsSlice, kind.String())
	}

	msg := fmt.Sprintf("%s %s", s.StrsOr(supportedKindsSlice), s.PluralS(userconfig.KindKey, len(supportedKindsSlice)))

	return errors.WithStack(&errors.Error{
		Kind:    ErrOperationIsOnlySupportedForKind,
		Message: fmt.Sprintf("this operation is only allowed for %s and is not supported for %s of kind %s", msg, resource.Name, resource.Kind),
	})
}

func ErrorAPINotDeployed(apiName string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrAPINotDeployed,
		Message: fmt.Sprintf("%s is not deployed", apiName),
	})
}

func ErrorAPIIDNotFound(apiName string, apiID string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrAPIIDNotFound,
		Message: fmt.Sprintf("%s with id %s has never been deployed", apiName, apiID),
	})
}

func ErrorCannotChangeKindOfDeployedAPI(name string, newKind, prevKind userconfig.Kind) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrCannotChangeTypeOfDeployedAPI,
		Message: fmt.Sprintf("cannot change the kind of %s to %s because it has already been deployed with kind %s; please delete it with `cortex delete %s` and redeploy after updating the api configuration appropriately", name, newKind.String(), prevKind.String(), name),
	})
}

func ErrorNoAvailableNodeComputeLimit(api *userconfig.API, compute userconfig.Compute, maxMemMap map[string]kresource.Quantity) error {
	msg := "no instance types in your cluster are large enough to satisfy the requested resources for your pod\n\n"
	msg += console.Bold("requested pod resources\n")
	msg += podResourceRequestsTable(api, compute)
	msg += "\n" + s.TrimTrailingNewLines(nodeGroupResourcesTable(api, compute, maxMemMap))

	return errors.WithStack(&errors.Error{
		Kind:    ErrNoAvailableNodeComputeLimit,
		Message: msg,
	})
}

func ErrorAPIUsedByTrafficSplitter(trafficSplitters []string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrRealtimeAPIUsedByTrafficSplitter,
		Message: fmt.Sprintf("cannot delete api because it is used by the following %s: %s", s.PluralS("TrafficSplitter", len(trafficSplitters)), s.StrsSentence(trafficSplitters, "")),
	})
}

func ErrorAPIsNotDeployed(notDeployedAPIs []string) error {
	message := fmt.Sprintf("apis %s were either not found or are not RealtimeAPIs", s.StrsAnd(notDeployedAPIs))
	if len(notDeployedAPIs) == 1 {
		message = fmt.Sprintf("api %s was either not found or is not a RealtimeAPI", notDeployedAPIs[0])
	}
	return errors.WithStack(&errors.Error{
		Kind:    ErrAPIsNotDeployed,
		Message: message,
	})
}

func ErrorInvalidNodeGroupSelector(selected string, availableNodeGroups []string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidNodeGroupSelector,
		Message: fmt.Sprintf("node group \"%s\" doesn't exist; remove the node group selector to let Cortex determine automatically where to place the API, or specify a valid node group name (%s)", selected, s.StrsOr(availableNodeGroups)),
	})
}

func ErrorNoNodeGroups() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrNoNodeGroups,
		Message: fmt.Sprintf("your api cannot be deployed because your cluster doesn't have any node groups; create a node group with `cortex cluster configure CLUSTER_CONFIG_FILE`"),
	})
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
