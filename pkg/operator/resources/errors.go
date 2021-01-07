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

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/strings"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
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

func ErrorNoAvailableNodeComputeLimit(resource string, reqStr string, maxStr string) error {
	message := fmt.Sprintf("no instances can satisfy the requested %s quantity - requested %s %s but instances only have %s %s available", resource, reqStr, resource, maxStr, resource)
	if maxStr == "0" {
		message = fmt.Sprintf("no instances can satisfy the requested %s quantity - requested %s %s but instances don't have any %s", resource, reqStr, resource, resource)
	}
	return errors.WithStack(&errors.Error{
		Kind:    ErrNoAvailableNodeComputeLimit,
		Message: message,
	})
}

func ErrorAPIUsedByTrafficSplitter(trafficSplitters []string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrRealtimeAPIUsedByTrafficSplitter,
		Message: fmt.Sprintf("cannot delete api because it is used by the following %s: %s", strings.PluralS("TrafficSplitter", len(trafficSplitters)), strings.StrsSentence(trafficSplitters, "")),
	})
}

func ErrorAPIsNotDeployed(notDeployedAPIs []string) error {
	message := fmt.Sprintf("apis %s were either not found or are not RealtimeAPIs", strings.StrsAnd(notDeployedAPIs))
	if len(notDeployedAPIs) == 1 {
		message = fmt.Sprintf("api %s was either not found or is not a RealtimeAPI", notDeployedAPIs[0])
	}
	return errors.WithStack(&errors.Error{
		Kind:    ErrAPIsNotDeployed,
		Message: message,
	})
}
