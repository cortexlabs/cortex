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

package cliconfig

import (
	"fmt"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
)

const (
	ErrEnvironmentNotConfigured     = "cliconfig.environment_not_configured"
	ErrEnvironmentAlreadyConfigured = "cliconfig.environment_already_configured"
	ErrDuplicateEnvironmentNames    = "cliconfig.duplicate_environment_names"
)

func ErrorEnvironmentNotConfigured(envName string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrEnvironmentNotConfigured,
		Message: fmt.Sprintf("there is no environment named %s", envName),
	})
}

func ErrorEnvironmentAlreadyConfigured(envName string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrEnvironmentAlreadyConfigured,
		Message: fmt.Sprintf("there is already an environment named %s", envName),
	})
}

func ErrorDuplicateEnvironmentNames(envName string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrDuplicateEnvironmentNames,
		Message: fmt.Sprintf("duplicate environment names (%s is defined more than once)", s.UserStr(envName)),
	})
}
