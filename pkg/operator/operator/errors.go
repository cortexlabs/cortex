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
	"fmt"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
)

const (
	ErrCortexInstallationBroken    = "operator.cortex_installation_broken"
	ErrLoadBalancerInitializing    = "operator.load_balancer_initializing"
	ErrMalformedConfig             = "operator.malformed_config"
	ErrNoAPIs                      = "operator.no_apis"
	ErrAPIUpdating                 = "operator.api_updating"
	ErrAPINotDeployed              = "operator.api_not_deployed"
	ErrNoAvailableNodeComputeLimit = "operator.no_available_node_compute_limit"
)

func ErrorCortexInstallationBroken() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrCortexInstallationBroken,
		Message: "cortex is out of date or not installed properly; run `cortex cluster update`, or spin down your cluster with `cortex cluster down` and create a new one with `cortex cluster up`",
	})
}

func ErrorLoadBalancerInitializing() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrLoadBalancerInitializing,
		Message: "load balancer is still initializing",
	})
}

func ErrorAPIUpdating(apiName string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrAPIUpdating,
		Message: fmt.Sprintf("%s is updating (override with --force)", apiName),
	})
}

func ErrorAPINotDeployed(apiName string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrAPINotDeployed,
		Message: fmt.Sprintf("%s is not deployed", apiName), // note: if modifying this string, search the codebase for it and change all occurrences
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
