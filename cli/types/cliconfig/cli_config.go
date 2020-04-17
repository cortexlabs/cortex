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
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/types"
)

type CLIConfig struct {
	Telemetry          *bool          `json:"telemetry,omitempty" yaml:"telemetry,omitempty"`
	DefaultEnvironment string         `json:"default_environment" yaml:"default_environment"`
	Environments       []*Environment `json:"environments" yaml:"environments"`
}

func (cliConfig CLIConfig) GetEnv(envName string) (Environment, error) {
	for _, env := range cliConfig.Environments {
		if env.Name == envName {
			return *env, nil
		}
	}

	return Environment{}, ErrorEnvironmentNotConfigured(envName)
}

func (cliConfig CLIConfig) GetDefaultEnv() (Environment, error) {
	return cliConfig.GetEnv(cliConfig.DefaultEnvironment)
}

func (cliConfig *CLIConfig) Validate() error {
	envNames := strset.New()

	for _, env := range cliConfig.Environments {
		if envNames.Has(env.Name) {
			return errors.Wrap(ErrorDuplicateEnvironmentNames(env.Name), EnvironmentsKey)
		}

		envNames.Add(env.Name)

		if err := env.Validate(); err != nil {
			return errors.Wrap(err, EnvironmentsKey)
		}
	}

	// Ensure the local env is always present
	if !envNames.Has(types.LocalProviderType.String()) {
		localEnv := &Environment{
			Name:     types.LocalProviderType.String(),
			Provider: types.LocalProviderType,
		}

		cliConfig.Environments = append([]*Environment{localEnv}, cliConfig.Environments...)
		envNames.Add("local")
	}

	if cliConfig.DefaultEnvironment == "" {
		cliConfig.DefaultEnvironment = types.LocalProviderType.String()
	}

	if !envNames.Has(cliConfig.DefaultEnvironment) {
		return ErrorDefaultEnvironmentDoesntExist(cliConfig.DefaultEnvironment)
	}

	return nil
}
