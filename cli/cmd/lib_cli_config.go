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

	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/prompt"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/urls"
	"github.com/cortexlabs/yaml"
)

type CLIConfig struct {
	Telemetry    bool           `json:"telemetry" yaml:"telemetry"`
	Environments []*Environment `json:"environments" yaml:"environments"`
}

type Environment struct {
	Name               string   `json:"name" yaml:"name"`
	Provider           Provider `json:"provider" yaml:"provider"`
	OperatorEndpoint   *string  `json:"operator_endpoint,omitempty" yaml:"operator_endpoint,omitempty"`
	AWSAccessKeyID     *string  `json:"aws_access_key_id,omitempty" yaml:"aws_access_key_id,omitempty"`
	AWSSecretAccessKey *string  `json:"aws_secret_access_key,omitempty" yaml:"aws_secret_access_key,omitempty"`
}

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
			StructField: "Environments",
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
								AllowedValues: ProviderStrings(),
							},
							Parser: func(str string) (interface{}, error) {
								return ProviderFromString(str), nil
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

func (cliConfig *CLIConfig) validate() error {
	envNames := strset.New()

	for _, env := range cliConfig.Environments {
		if envNames.Has(env.Name) {
			return errors.Wrap(ErrorDuplicateEnvironmentNames(env.Name), "environments")
		}

		envNames.Add(env.Name)

		if err := env.validate(); err != nil {
			return errors.Wrap(err, "environments")
		}
	}

	// Ensure the local env is always present
	if !envNames.Has(Local.String()) {
		localEnv := &Environment{
			Name:     Local.String(),
			Provider: Local,
		}

		cliConfig.Environments = append([]*Environment{localEnv}, cliConfig.Environments...)
	}

	return nil
}

func (env *Environment) validate() error {
	if env.Name == "" {
		return errors.Wrap(cr.ErrorMustBeDefined(), "name")
	}
	if env.Provider == UnknownProvider {
		return errors.Wrap(cr.ErrorMustBeDefined(ProviderStrings()), env.Name, "provider")
	}

	if err := checkReservedEnvironmentNames(env.Name, env.Provider); err != nil {
		return err
	}

	if env.Provider == Local {
		if env.OperatorEndpoint != nil {
			return errors.Wrap(ErrorOperatorEndpointInLocalEnvironment(), env.Name)
		}
	}

	if env.Provider == AWS {
		if env.OperatorEndpoint == nil {
			return errors.Wrap(cr.ErrorMustBeDefined(), env.Name, "operator_endpoint")
		}
		if env.AWSAccessKeyID == nil {
			return errors.Wrap(cr.ErrorMustBeDefined(), env.Name, "aws_access_key_id")
		}
		if env.AWSSecretAccessKey == nil {
			return errors.Wrap(cr.ErrorMustBeDefined(), env.Name, "aws_secret_access_key")
		}
	}

	return nil
}

func checkReservedEnvironmentNames(envName string, provider Provider) error {
	envNameProvider := ProviderFromString(envName)
	if envNameProvider == UnknownProvider {
		return nil
	}

	if envNameProvider != provider {
		return ErrorEnvironmentProviderNameConflict(envName, provider)
	}

	return nil
}

func providerPromptValidation(envName string, defaults Environment) *cr.PromptValidation {
	defaultProviderStr := ""
	if defaults.Provider != UnknownProvider {
		defaultProviderStr = defaults.Provider.String()
	}

	return &cr.PromptValidation{
		SkipNonEmptyFields: true,
		PromptItemValidations: []*cr.PromptItemValidation{
			{
				StructField: "Provider",
				PromptOpts: &prompt.Options{
					Prompt: fmt.Sprintf("provider (%s)", s.StrsOr(ProviderStrings())),
				},
				StringValidation: &cr.StringValidation{
					Required:      true,
					Default:       defaultProviderStr,
					AllowedValues: ProviderStrings(),
				},
				Parser: func(str string) (interface{}, error) {
					provider := ProviderFromString(str)
					if err := checkReservedEnvironmentNames(envName, provider); err != nil {
						return nil, err
					}
					return provider, nil
				},
			},
		},
	}
}

func localEnvPromptValidation(defaults Environment) *cr.PromptValidation {
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

func awsEnvPromptValidation(defaults Environment) *cr.PromptValidation {
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

// Will return nil if not configured, except for local
func readEnv(envName string) (*Environment, error) {
	cliConfig, err := readCLIConfig()
	if err != nil {
		return nil, err
	}

	for _, env := range cliConfig.Environments {
		if env.Name == envName {
			return env, nil
		}
	}

	if envName == Local.String() {
		return &Environment{
			Name:     Local.String(),
			Provider: Local,
		}, nil
	}

	return nil, nil
}

func readOrConfigureNonLocalEnv(envName string) (Environment, error) {
	env, err := readEnv(envName)
	if err != nil {
		return Environment{}, err
	}

	if env != nil {
		if env.Provider == Local {
			return Environment{}, ErrorLocalProviderNotSupported(*env)
		}
		return *env, nil
	}

	promptStr := fmt.Sprintf("the %s environment is not configured; do you already have a Cortex cluster running on AWS?", envName)
	yesMsg := fmt.Sprintf("please configure the %s environment to point to your running cluster:\n", envName)
	noMsg := "you can create a cluster on AWS by running the `cortex cluster up` command"
	prompt.YesOrExit(promptStr, yesMsg, noMsg)

	fieldsToSkipPrompt := Environment{
		Provider: AWS,
	}
	return configureEnv(envName, fieldsToSkipPrompt)
}

func getDefaultEnvConfig(envName string) Environment {
	defaults := Environment{}

	prevEnv, err := readEnv(envName)
	if err == nil && prevEnv != nil {
		defaults = *prevEnv
	}

	if defaults.Provider == UnknownProvider {
		if envNameProvider := ProviderFromString(envName); envNameProvider != UnknownProvider {
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

func configureEnv(envName string, fieldsToSkipPrompt Environment) (Environment, error) {
	fmt.Println("environment: " + envName + "\n")

	defaults := getDefaultEnvConfig(envName)

	if fieldsToSkipPrompt.Provider == UnknownProvider {
		if envNameProvider := ProviderFromString(envName); envNameProvider != UnknownProvider {
			fieldsToSkipPrompt.Provider = envNameProvider
		}
	}

	env := Environment{
		Name:               envName,
		Provider:           fieldsToSkipPrompt.Provider,
		OperatorEndpoint:   fieldsToSkipPrompt.OperatorEndpoint,
		AWSAccessKeyID:     fieldsToSkipPrompt.AWSAccessKeyID,
		AWSSecretAccessKey: fieldsToSkipPrompt.AWSSecretAccessKey,
	}

	err := cr.ReadPrompt(&env, providerPromptValidation(envName, defaults))
	if err != nil {
		return Environment{}, err
	}

	switch env.Provider {
	case Local:
		err = cr.ReadPrompt(&env, localEnvPromptValidation(defaults))
	case AWS:
		err = cr.ReadPrompt(&env, awsEnvPromptValidation(defaults))
	}
	if err != nil {
		return Environment{}, err
	}

	if err := env.validate(); err != nil {
		return Environment{}, err
	}

	if err := addEnvToCLIConfig(env); err != nil {
		return Environment{}, err
	}

	return env, nil
}

func readCLIConfig() (CLIConfig, error) {
	if !files.IsFile(_cliConfigPath) {
		// add empty file so that the file created by the manager container maintains current user permissions
		files.MakeEmptyFile(_cliConfigPath)

		return CLIConfig{
			Telemetry: true,
		}, nil
	}

	cliConfig := CLIConfig{}
	errs := cr.ParseYAMLFile(&cliConfig, _cliConfigValidation, _cliConfigPath)
	if errors.HasError(errs) {
		return CLIConfig{}, errors.FirstError(errs...)
	}

	if err := cliConfig.validate(); err != nil {
		return CLIConfig{}, errors.Wrap(err, _cliConfigPath)
	}

	return cliConfig, nil
}

func addEnvToCLIConfig(newEnv Environment) error {
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

	cliConfig.validate()

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

	var updatedEnvs []*Environment
	deleted := false
	for _, env := range cliConfig.Environments {
		if env.Name == envName {
			deleted = true
			continue
		}
		updatedEnvs = append(updatedEnvs, env)
	}

	if deleted == false && envName != Local.String() {
		return ErrorEnvironmentNotConfigured(envName)
	}

	cliConfig.Environments = updatedEnvs

	cliConfig.validate()

	cliConfigBytes, err := yaml.Marshal(cliConfig)
	if err != nil {
		return errors.WithStack(err)
	}
	if err := files.WriteFile(cliConfigBytes, _cliConfigPath); err != nil {
		return err
	}

	return nil
}
