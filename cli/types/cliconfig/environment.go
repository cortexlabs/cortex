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
	"strings"

	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/console"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/urls"
)

type Environment struct {
	Name             string `json:"name" yaml:"name"`
	OperatorEndpoint string `json:"operator_endpoint" yaml:"operator_endpoint"`
}

func (env Environment) String(isDefault bool) string {
	var envStr string

	if isDefault {
		envStr += console.Bold(env.Name + " (default)")
	} else {
		envStr += console.Bold(env.Name)
	}

	envStr += fmt.Sprintf("\ncortex operator endpoint: %s\n", env.OperatorEndpoint)

	return envStr
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
