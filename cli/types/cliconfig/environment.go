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
	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/table"
	"github.com/cortexlabs/cortex/pkg/types"
)

type Environment struct {
	Name               string             `json:"name" yaml:"name"`
	Provider           types.ProviderType `json:"provider" yaml:"provider"`
	OperatorEndpoint   *string            `json:"operator_endpoint,omitempty" yaml:"operator_endpoint,omitempty"`
	AWSAccessKeyID     *string            `json:"aws_access_key_id,omitempty" yaml:"aws_access_key_id,omitempty"`
	AWSSecretAccessKey *string            `json:"aws_secret_access_key,omitempty" yaml:"aws_secret_access_key,omitempty"`
}

func (env Environment) String(isDefault bool) string {
	var items table.KeyValuePairs

	if isDefault {
		items.Add("name", env.Name+" (default)")
	} else {
		items.Add("name", env.Name)
	}

	items.Add("provider", env.Provider)

	if env.OperatorEndpoint != nil {
		items.Add("cortex operator endpoint", *env.OperatorEndpoint)
	}
	if env.AWSAccessKeyID != nil {
		items.Add("aws access key id", *env.AWSAccessKeyID)
	}
	if env.AWSSecretAccessKey != nil {
		items.Add("aws secret access key", s.MaskString(*env.AWSSecretAccessKey, 4))
	}

	return items.String(&table.KeyValuePairOpts{
		BoldFirstLine: pointer.Bool(true),
	})
}

func CheckProviderEnvironmentNameCompatibility(envName string, provider types.ProviderType) error {
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

	if err := CheckProviderEnvironmentNameCompatibility(env.Name, env.Provider); err != nil {
		return err
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

	if env.Provider == types.GCPProviderType {
		if env.OperatorEndpoint == nil {
			return errors.Wrap(cr.ErrorMustBeDefined(), env.Name, OperatorEndpointKey)
		}
		if env.AWSAccessKeyID != nil {
			return errors.Wrap(cr.ErrorMustBeEmpty(), env.Name, AWSAccessKeyIDKey)
		}
		if env.AWSSecretAccessKey != nil {
			return errors.Wrap(cr.ErrorMustBeEmpty(), env.Name, AWSSecretAccessKeyKey)
		}
	}

	return nil
}
