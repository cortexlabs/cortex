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

package operator

import (
	"fmt"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
)

const (
	ErrCortexInstallationBroken = "operator.cortex_installation_broken"
	ErrLoadBalancerInitializing = "operator.load_balancer_initializing"
	ErrInvalidOperatorLogLevel  = "operator.invalid_operator_log_level"
)

func ErrorCortexInstallationBroken() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrCortexInstallationBroken,
		Message: "cortex is out of date or not installed properly; run `cortex cluster configure` to repair, or spin down your cluster with `cortex cluster down` and create a new one with `cortex cluster up`",
	})
}

func ErrorLoadBalancerInitializing() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrLoadBalancerInitializing,
		Message: "load balancer is still initializing",
	})
}

func ErrorInvalidOperatorLogLevel(provided string, loglevels []string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrLoadBalancerInitializing,
		Message: fmt.Sprintf("invalid operator log level %s; must be one of %s", provided, s.StrsOr(loglevels)),
	})
}
