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

package cliconfig

import (
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
)

type CLIConfig struct {
	Telemetry          *bool          `json:"telemetry,omitempty" yaml:"telemetry,omitempty"`
	DefaultEnvironment *string        `json:"default_environment" yaml:"default_environment"`
	Environments       []*Environment `json:"environments" yaml:"environments"`
}

type UserFacingCLIConfig struct {
	DefaultEnvironment *string        `json:"default_environment" yaml:"default_environment"`
	Environments       []*Environment `json:"environments" yaml:"environments"`
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

	// Backwards compatibility: ignore local default env
	defaultEnv := cliConfig.DefaultEnvironment
	if defaultEnv != nil && *defaultEnv == "local" && !envNames.Has(*defaultEnv) {
		cliConfig.DefaultEnvironment = nil
	}

	return nil
}

func (cliConfig *CLIConfig) ConvertToUserFacingCLIConfig() UserFacingCLIConfig {
	return UserFacingCLIConfig{
		DefaultEnvironment: cliConfig.DefaultEnvironment,
		Environments:       cliConfig.Environments,
	}
}
