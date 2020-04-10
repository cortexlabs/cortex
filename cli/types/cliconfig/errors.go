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

package cliconfig

import (
	"fmt"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/types"
)

const (
	ErrEnvironmentNotConfigured           = "cliconfig.environment_not_configured"
	ErrDuplicateEnvironmentNames          = "cliconfig.duplicate_environment_names"
	ErrEnvironmentProviderNameConflict    = "cliconfig.environment_provider_name_conflict"
	ErrOperatorEndpointInLocalEnvironment = "cliconfig.operator_endpoint_in_local_environment"
)

func ErrorEnvironmentNotConfigured(envName string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrEnvironmentNotConfigured,
		Message: fmt.Sprintf("%s environment is not configured", envName),
	})
}

func ErrorEnvironmentProviderNameConflict(envName string, provider types.ProviderType) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrEnvironmentProviderNameConflict,
		Message: fmt.Sprintf("the %s environment cannot use the %s provider", envName, provider.String()),
	})
}

func ErrorDuplicateEnvironmentNames(envName string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrDuplicateEnvironmentNames,
		Message: fmt.Sprintf("duplicate environment names (%s is defined more than once)", s.UserStr(envName)),
	})
}

func ErrorOperatorEndpointInLocalEnvironment() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrOperatorEndpointInLocalEnvironment,
		Message: fmt.Sprintf("operator_endpoint should not be specified (it's not used in local providers)"),
	})
}
