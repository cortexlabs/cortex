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
	"strings"

	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/table"
	"github.com/cortexlabs/cortex/pkg/lib/urls"
	"github.com/cortexlabs/cortex/pkg/types"
)

type Environment struct {
	Name             string             `json:"name" yaml:"name"`
	Provider         types.ProviderType `json:"provider" yaml:"provider"`
	OperatorEndpoint string             `json:"operator_endpoint" yaml:"operator_endpoint"`
}

func (env Environment) String(isDefault bool) string {
	var items table.KeyValuePairs

	if isDefault {
		items.Add("name", env.Name+" (default)")
	} else {
		items.Add("name", env.Name)
	}

	items.Add("provider", env.Provider)
	items.Add("cortex operator endpoint", env.OperatorEndpoint)

	return items.String(&table.KeyValuePairOpts{
		BoldFirstLine: pointer.Bool(true),
	})
}

func CortexEndpointValidator(val string) (string, error) {
	urlStr := strings.TrimSpace(val)

	parsedURL, err := urls.Parse(urlStr)
	if err != nil {
		return "", err
	}

	// default https
	if parsedURL.Scheme == "" {
		parsedURL.Scheme = "https"
	}

	return parsedURL.String(), nil
}

func (env *Environment) Validate() error {
	if env.Name == "" {
		return errors.Wrap(cr.ErrorMustBeDefined(), NameKey)
	}

	validOperatorURL, err := CortexEndpointValidator(env.OperatorEndpoint)
	if err != nil {
		return err
	}

	env.OperatorEndpoint = validOperatorURL
	return nil
}
