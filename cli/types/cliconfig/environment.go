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
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/types"
)

type Environment struct {
	Name               string             `json:"name" yaml:"name"`
	Provider           types.ProviderType `json:"provider" yaml:"provider"`
	OperatorEndpoint   *string            `json:"operator_endpoint,omitempty" yaml:"operator_endpoint,omitempty"`
	AWSAccessKeyID     *string            `json:"aws_access_key_id,omitempty" yaml:"aws_access_key_id,omitempty"`
	AWSSecretAccessKey *string            `json:"aws_secret_access_key,omitempty" yaml:"aws_secret_access_key,omitempty"`
	AWSRegion          *string            `json:"aws_region,omitempty" yaml:"aws_region,omitempty"`
}

func CheckReservedEnvironmentNames(envName string, provider types.ProviderType) error {
	envNameProvider := types.ProviderTypeFromString(envName)
	if envNameProvider == types.UnknownProviderType {
		return nil
	}

	if envNameProvider != provider {
		return ErrorEnvironmentProviderNameConflict(envName, provider)
	}

	return nil
}

func (env *Environment) Validate() error {
	if env.Name == "" {
		return errors.Wrap(cr.ErrorMustBeDefined(), NameKey)
	}

	if env.Provider == types.UnknownProviderType {
		return errors.Wrap(cr.ErrorMustBeDefined(types.ProviderTypeStrings()), env.Name, ProviderKey)
	}

	if err := CheckReservedEnvironmentNames(env.Name, env.Provider); err != nil {
		return err
	}

	if env.Provider == types.LocalProviderType {
		if env.OperatorEndpoint != nil {
			return errors.Wrap(ErrorOperatorEndpointInLocalEnvironment(), env.Name)
		}
	}

	if env.Provider == types.AWSProviderType {
		if env.OperatorEndpoint == nil {
			return errors.Wrap(cr.ErrorMustBeDefined(), env.Name, OperatorEndpointKey)
		}
		if env.AWSAccessKeyID == nil {
			return errors.Wrap(cr.ErrorMustBeDefined(), env.Name, AWSAccessKeyIDKey)
		}
		if env.AWSSecretAccessKey == nil {
			return errors.Wrap(cr.ErrorMustBeDefined(), env.Name, AWSSecretAccessKeyKey)
		}
	}

	if env.AWSRegion != nil {
		if !aws.S3Regions.Has(*env.AWSRegion) {
			s3RegionSlice := aws.S3Regions.Slice()
			return errors.Wrap(cr.ErrorInvalidStr(*env.AWSRegion, s3RegionSlice[0], s3RegionSlice[1:]...), env.Name, AWSRegionKey)
		}
	}

	return nil
}
