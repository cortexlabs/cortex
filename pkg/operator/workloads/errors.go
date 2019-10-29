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
	"fmt"

	s "github.com/cortexlabs/cortex/pkg/lib/strings"
)

type ErrorKind int

const (
	ErrUnknown ErrorKind = iota
	ErrMoreThanOneWorkflow
	ErrCortexInstallationBroken
	ErrLoadBalancerInitializing
	ErrNotFound
	ErrAPIInitializing
	ErrNoAvailableNodeComputeLimit
	ErrDuplicateEndpointOtherDeployment
)

var errorKinds = []string{
	"err_unknown",
	"err_more_than_one_workflow",
	"err_cortex_installation_broken",
	"err_load_balancer_initializing",
	"err_not_found",
	"err_api_initializing",
	"err_no_available_node_compute_limit",
	"err_duplicate_endpoint_other_deployment",
}

var _ = [1]int{}[int(ErrDuplicateEndpointOtherDeployment)-(len(errorKinds)-1)] // Ensure list length matches

func (t ErrorKind) String() string {
	return errorKinds[t]
}

// MarshalText satisfies TextMarshaler
func (t ErrorKind) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// UnmarshalText satisfies TextUnmarshaler
func (t *ErrorKind) UnmarshalText(text []byte) error {
	enum := string(text)
	for i := 0; i < len(errorKinds); i++ {
		if enum == errorKinds[i] {
			*t = ErrorKind(i)
			return nil
		}
	}

	*t = ErrUnknown
	return nil
}

// UnmarshalBinary satisfies BinaryUnmarshaler
// Needed for msgpack
func (t *ErrorKind) UnmarshalBinary(data []byte) error {
	return t.UnmarshalText(data)
}

// MarshalBinary satisfies BinaryMarshaler
func (t ErrorKind) MarshalBinary() ([]byte, error) {
	return []byte(t.String()), nil
}

type Error struct {
	Kind    ErrorKind
	message string
}

func (e Error) Error() string {
	return e.message
}

func ErrorMoreThanOneWorkflow() error {
	return Error{
		Kind:    ErrMoreThanOneWorkflow,
		message: "there is more than one workflow",
	}
}

func ErrorCortexInstallationBroken() error {
	return Error{
		Kind:    ErrCortexInstallationBroken,
		message: "cortex is out of date, or not installed properly on your cluster; run `cortex cluster update`",
	}
}

func ErrorLoadBalancerInitializing() error {
	return Error{
		Kind:    ErrLoadBalancerInitializing,
		message: "load balancer is still initializing",
	}
}

func ErrorNotFound() error {
	return Error{
		Kind:    ErrNotFound,
		message: "not found",
	}
}

func ErrorAPIInitializing() error {
	return Error{
		Kind:    ErrAPIInitializing,
		message: "api is still initializing",
	}
}

func ErrorNoAvailableNodeComputeLimit(resource string, reqStr string, maxStr string) error {
	message := fmt.Sprintf("no available nodes can satisfy the requested %s quantity - requested %s %s but nodes only have %s %s", resource, reqStr, resource, maxStr, resource)
	if maxStr == "0" {
		message = fmt.Sprintf("no available nodes can satisfy the requested %s quantity - requested %s %s but nodes don't have any %s", resource, reqStr, resource, resource)
	}
	return Error{
		Kind:    ErrNoAvailableNodeComputeLimit,
		message: message,
	}
}

func ErrorDuplicateEndpointOtherDeployment(appName string, apiName string) error {
	return Error{
		Kind:    ErrDuplicateEndpointOtherDeployment,
		message: fmt.Sprintf("endpoint is already in use by an API named %s in the %s deployment", s.UserStr(apiName), s.UserStr(appName)),
	}
}
