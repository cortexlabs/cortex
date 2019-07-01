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

type ErrorKind int

const (
	ErrUnknown ErrorKind = iota
	ErrMoreThanOneWorkflow
	ErrContextAppMismatch
	ErrWorkflowAppMismatch
	ErrCortexInstallationBroken
	ErrLoadBalancerInitializing
	ErrNotFound
)

var errorKinds = []string{
	"err_unknown",
	"err_more_than_one_workflow",
	"err_context_app_mismatch",
	"err_workflow_app_mismatch",
	"err_cortex_installation_broken",
	"err_load_balancer_initializing",
	"err_not_found",
}

var _ = [1]int{}[int(ErrNotFound)-(len(errorKinds)-1)] // Ensure list length matches

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

func ErrorContextAppMismatch() error {
	return Error{
		Kind:    ErrContextAppMismatch,
		message: "context deployments do not match",
	}
}

func ErrorWorkflowAppMismatch() error {
	return Error{
		Kind:    ErrWorkflowAppMismatch,
		message: "workflow deployments do not match",
	}
}

func ErrorCortexInstallationBroken() error {
	return Error{
		Kind:    ErrCortexInstallationBroken,
		message: "cortex is out of date, or not installed properly on your cluster; run `./cortex.sh update`",
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
