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

package cmd

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"

	"github.com/cortexlabs/cortex/cli/cluster"
	"github.com/cortexlabs/cortex/cli/types/cliconfig"
	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/prompt"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/urls"
	"github.com/cortexlabs/cortex/pkg/types"
	"github.com/cortexlabs/yaml"
)

var _cliConfigValidation = &cr.StructValidation{
	TreatNullAsEmpty: true,
	StructFieldValidations: []*cr.StructFieldValidation{
		{
			StructField: "Telemetry",
			BoolValidation: &cr.BoolValidation{
				Default:  true,
				Required: false,
			},
		},
		{
			StructField: "cliconfig.Environments",
			StructListValidation: &cr.StructListValidation{
				AllowExplicitNull: true,
				StructValidation: &cr.StructValidation{
					StructFieldValidations: []*cr.StructFieldValidation{
						{
							StructField: "Name",
							StringValidation: &cr.StringValidation{
								Required:  true,
								MaxLength: 63,
							},
						},
						{
							StructField: "Provider",
							StringValidation: &cr.StringValidation{
								Required:      true,
								AllowedValues: types.ProviderTypeStrings(),
							},
							Parser: func(str string) (interface{}, error) {
								return types.ProviderTypeFromString(str), nil
							},
						},
						{
							StructField: "OperatorEndpoint",
							StringPtrValidation: &cr.StringPtrValidation{
								Required:  false,
								Validator: cr.GetURLValidator(false, false),
							},
						},
						{
							StructField: "AWSAccessKeyID",
							StringPtrValidation: &cr.StringPtrValidation{
								Required: false,
							},
						},
						{
							StructField: "AWSSecretAccessKey",
							StringPtrValidation: &cr.StringPtrValidation{
								Required: false,
							},
						},
					},
				},
			},
		},
	},
}

func providerPromptValidation(envName string, defaults cliconfig.Environment) *cr.PromptValidation {
	defaultProviderStr := ""
	if defaults.Provider != types.UnknownProviderType {
		defaultProviderStr = defaults.Provider.String()
	}

	return &cr.PromptValidation{
		SkipNonEmptyFields: true,
		PromptItemValidations: []*cr.PromptItemValidation{
			{
				StructField: "Provider",
				PromptOpts: &prompt.Options{
					Prompt: fmt.Sprintf("provider (%s)", s.StrsOr(types.ProviderTypeStrings())),
				},
				StringValidation: &cr.StringValidation{
					Required:      true,
					Default:       defaultProviderStr,
					AllowedValues: types.ProviderTypeStrings(),
				},
				Parser: func(str string) (interface{}, error) {
					provider := types.ProviderTypeFromString(str)
					if err := cliconfig.CheckReservedEnvironmentNames(envName, provider); err != nil {
						return nil, err
					}
					return provider, nil
				},
			},
		},
	}
}

func localEnvPromptValidation(defaults cliconfig.Environment) *cr.PromptValidation {
	accessKeyIDPrompt := "aws access key id"
	if defaults.AWSAccessKeyID == nil {
		accessKeyIDPrompt += " [press ENTER to skip]"
	}

	secretAccessKeyPrompt := "aws secret access key"
	if defaults.AWSSecretAccessKey == nil {
		secretAccessKeyPrompt += " [press ENTER to skip]"
	}

	return &cr.PromptValidation{
		SkipNonEmptyFields: true,
		PromptItemValidations: []*cr.PromptItemValidation{
			{
				StructField: "AWSAccessKeyID",
				PromptOpts: &prompt.Options{
					Prompt: accessKeyIDPrompt,
				},
				StringPtrValidation: &cr.StringPtrValidation{
					Required:   false,
					AllowEmpty: true,
					Default:    defaults.AWSAccessKeyID,
				},
			},
			{
				StructField: "AWSSecretAccessKey",
				PromptOpts: &prompt.Options{
					Prompt:      secretAccessKeyPrompt,
					MaskDefault: true,
					HideTyping:  true,
				},
				StringPtrValidation: &cr.StringPtrValidation{
					Required:   false,
					AllowEmpty: true,
					Default:    defaults.AWSSecretAccessKey,
				},
			},
		},
	}
}

func awsEnvPromptValidation(defaults cliconfig.Environment) *cr.PromptValidation {
	return &cr.PromptValidation{
		SkipNonEmptyFields: true,
		PromptItemValidations: []*cr.PromptItemValidation{
			{
				StructField: "OperatorEndpoint",
				PromptOpts: &prompt.Options{
					Prompt: "cortex operator endpoint",
				},
				StringPtrValidation: &cr.StringPtrValidation{
					Required:  true,
					Default:   defaults.OperatorEndpoint,
					Validator: validateOperatorEndpoint,
				},
			},
			{
				StructField: "AWSAccessKeyID",
				PromptOpts: &prompt.Options{
					Prompt: "aws access key id",
				},
				StringPtrValidation: &cr.StringPtrValidation{
					Required: true,
					Default:  defaults.AWSAccessKeyID,
				},
			},
			{
				StructField: "AWSSecretAccessKey",
				PromptOpts: &prompt.Options{
					Prompt:      "aws secret access key",
					MaskDefault: true,
					HideTyping:  true,
				},
				StringPtrValidation: &cr.StringPtrValidation{
					Required: true,
					Default:  defaults.AWSSecretAccessKey,
				},
			},
		},
	}
}

// Only validate this during prompt, not when reading from file
func validateOperatorEndpoint(endpoint string) (string, error) {
	url, err := cr.GetURLValidator(false, false)(endpoint)
	if err != nil {
		return "", err
	}

	parsedURL, err := urls.Parse(url)
	if err != nil {
		return "", err
	}

	parsedURL.Scheme = "https"

	url = parsedURL.String()

	req, err := http.NewRequest("GET", urls.Join(url, "/verifycortex"), nil)
	if err != nil {
		return "", errors.Wrap(err, "verifying operator endpoint", url)
	}
	req.Header.Set("Content-Type", "application/json")

	client := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	response, err := client.Do(req)
	if err != nil {
		return "", ErrorInvalidOperatorEndpoint(url)
	}

	if response.StatusCode != 200 {
		return "", ErrorInvalidOperatorEndpoint(url)
	}

	return url, nil
}

func readTelemetryConfig() (bool, error) {
	cliConfig, err := readCLIConfig()
	if err != nil {
		return false, err
	}

	return cliConfig.Telemetry, nil
}

// Returns false if there is an error reading the CLI config
func isTelemetryEnabled() bool {
	enabled, err := readTelemetryConfig()
	if err != nil {
		return false
	}
	return enabled
}

func readCLIConfig() (cliconfig.CLIConfig, error) {
	if !files.IsFile(_cliConfigPath) {
		// add empty file so that the file created by the manager container maintains current user permissions
		files.MakeEmptyFile(_cliConfigPath)

		return cliconfig.CLIConfig{
			Telemetry: true,
		}, nil
	}

	cliConfig := cliconfig.CLIConfig{}
	errs := cr.ParseYAMLFile(&cliConfig, _cliConfigValidation, _cliConfigPath)
	if errors.HasError(errs) {
		return cliconfig.CLIConfig{}, errors.FirstError(errs...)
	}

	if err := cliConfig.Validate(); err != nil {
		return cliconfig.CLIConfig{}, errors.Wrap(err, _cliConfigPath)
	}

	return cliConfig, nil
}

func addEnvToCLIConfig(newEnv cliconfig.Environment) error {
	cliConfig, err := readCLIConfig()
	if err != nil {
		return err
	}

	replaced := false
	for i, prevEnv := range cliConfig.Environments {
		if prevEnv.Name == newEnv.Name {
			cliConfig.Environments[i] = &newEnv
			replaced = true
			break
		}
	}

	if !replaced {
		cliConfig.Environments = append(cliConfig.Environments, &newEnv)
	}

	err = cliConfig.Validate()
	if err != nil {
		return err
	}

	cliConfigBytes, err := yaml.Marshal(cliConfig)
	if err != nil {
		return errors.WithStack(err)
	}
	if err := files.WriteFile(cliConfigBytes, _cliConfigPath); err != nil {
		return err
	}

	return nil
}

func removeEnvFromCLIConfig(envName string) error {
	cliConfig, err := readCLIConfig()
	if err != nil {
		return err
	}

	var updatedEnvs []*cliconfig.Environment
	deleted := false
	for _, env := range cliConfig.Environments {
		if env.Name == envName {
			deleted = true
			continue
		}
		updatedEnvs = append(updatedEnvs, env)
	}

	if deleted == false && envName != types.LocalProviderType.String() {
		return cliconfig.ErrorEnvironmentNotConfigured(envName)
	}

	cliConfig.Environments = updatedEnvs

	err = cliConfig.Validate()
	if err != nil {
		return err
	}

	cliConfigBytes, err := yaml.Marshal(cliConfig)
	if err != nil {
		return errors.WithStack(err)
	}
	if err := files.WriteFile(cliConfigBytes, _cliConfigPath); err != nil {
		return err
	}

	return nil
}

// TODO Will return nil if not configured, except for local
func readEnv(envName string) (*cliconfig.Environment, error) {
	cliConfig, err := readCLIConfig()
	if err != nil {
		return nil, err
	}

	for _, env := range cliConfig.Environments {
		if env.Name == envName {
			return env, nil
		}
	}

	if envName == types.LocalProviderType.String() {
		return &cliconfig.Environment{
			Name:     types.LocalProviderType.String(),
			Provider: types.LocalProviderType,
		}, nil
	}

	return nil, nil
}

func MustReadOrConfigureEnv(envName string) cliconfig.Environment {
	existingEnv, err := readEnv(envName)
	if err != nil {
		exit.Error(err)
	}

	if existingEnv != nil {
		return *existingEnv
	}

	promptStr := fmt.Sprintf("the %s environment is not configured; do you already have a Cortex cluster running on AWS?", envName)
	yesMsg := fmt.Sprintf("please configure the %s environment to point to your running cluster:\n", envName)
	noMsg := "you can create a cluster on AWS by running the `cortex cluster up` command"
	prompt.YesOrExit(promptStr, yesMsg, noMsg)

	fieldsToSkipPrompt := cliconfig.Environment{
		Provider: types.AWSProviderType,
	}

	env, err := configureEnv(envName, fieldsToSkipPrompt)
	if err != nil {
		exit.Error(err)
	}
	return env
}

func ReadOrConfigureNonLocalEnv(envName string) (cliconfig.Environment, error) {
	env, err := readEnv(envName)
	if err != nil {
		return cliconfig.Environment{}, err
	}

	if env != nil {
		if env.Provider == types.LocalProviderType {
			return cliconfig.Environment{}, ErrorLocalProviderNotSupported(*env)
		}
		return *env, nil
	}

	promptStr := fmt.Sprintf("the %s environment is not configured; do you already have a Cortex cluster running on AWS?", envName)
	yesMsg := fmt.Sprintf("please configure the %s environment to point to your running cluster:\n", envName)
	noMsg := "you can create a cluster on AWS by running the `cortex cluster up` command"
	prompt.YesOrExit(promptStr, yesMsg, noMsg)

	fieldsToSkipPrompt := cliconfig.Environment{
		Provider: types.AWSProviderType,
	}
	return configureEnv(envName, fieldsToSkipPrompt)
}

func getDefaultEnvConfig(envName string) cliconfig.Environment {
	defaults := cliconfig.Environment{}

	prevEnv, err := readEnv(envName)
	if err == nil && prevEnv != nil {
		defaults = *prevEnv
	}

	if defaults.Provider == types.UnknownProviderType {
		if envNameProvider := types.ProviderTypeFromString(envName); envNameProvider != types.UnknownProviderType {
			defaults.Provider = envNameProvider
		}
	}

	if defaults.AWSAccessKeyID == nil && os.Getenv("AWS_ACCESS_KEY_ID") != "" {
		defaults.AWSAccessKeyID = pointer.String(os.Getenv("AWS_ACCESS_KEY_ID"))
	}
	if defaults.AWSSecretAccessKey == nil && os.Getenv("AWS_SECRET_ACCESS_KEY") != "" {
		defaults.AWSSecretAccessKey = pointer.String(os.Getenv("AWS_SECRET_ACCESS_KEY"))
	}
	if defaults.OperatorEndpoint == nil && os.Getenv("CORTEX_OPERATOR_ENDPOINT") != "" {
		defaults.OperatorEndpoint = pointer.String(os.Getenv("CORTEX_OPERATOR_ENDPOINT"))
	}

	return defaults
}

func configureEnv(envName string, fieldsToSkipPrompt cliconfig.Environment) (cliconfig.Environment, error) {
	fmt.Println("environment: " + envName + "\n")

	defaults := getDefaultEnvConfig(envName)

	if fieldsToSkipPrompt.Provider == types.UnknownProviderType {
		if envNameProvider := types.ProviderTypeFromString(envName); envNameProvider != types.UnknownProviderType {
			fieldsToSkipPrompt.Provider = envNameProvider
		}
	}

	env := cliconfig.Environment{
		Name:               envName,
		Provider:           fieldsToSkipPrompt.Provider,
		OperatorEndpoint:   fieldsToSkipPrompt.OperatorEndpoint,
		AWSAccessKeyID:     fieldsToSkipPrompt.AWSAccessKeyID,
		AWSSecretAccessKey: fieldsToSkipPrompt.AWSSecretAccessKey,
	}

	err := cr.ReadPrompt(&env, providerPromptValidation(envName, defaults))
	if err != nil {
		return cliconfig.Environment{}, err
	}

	switch env.Provider {
	case types.LocalProviderType:
		err = cr.ReadPrompt(&env, localEnvPromptValidation(defaults))
	case types.AWSProviderType:
		err = cr.ReadPrompt(&env, awsEnvPromptValidation(defaults))
	}
	if err != nil {
		return cliconfig.Environment{}, err
	}

	if err := env.Validate(); err != nil {
		return cliconfig.Environment{}, err
	}

	if err := addEnvToCLIConfig(env); err != nil {
		return cliconfig.Environment{}, err
	}

	return env, nil
}

func MustGetOperatorConfig(envName string) cluster.OperatorConfig {
	cliConfig, err := readCLIConfig()
	if err != nil {
		exit.Error(err)
	}
	clientID := clientID()
	env, err := cliConfig.GetEnv(envName)
	if err != nil {
		exit.Error(err)
	}

	return cluster.OperatorConfig{
		Telemetry:   isTelemetryEnabled(),
		ClientID:    clientID,
		Environment: env,
	}
}
