/*
Copyright 2019 Cortex Labs, Inc.

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
	"fmt"
	"os"

	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/prompt"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/yaml"
)

type CLIConfig struct {
	Telemetry    bool            `json:"telemetry" yaml:"telemetry"`
	Environments []*CLIEnvConfig `json:"environments" yaml:"environments"`
}

type CLIEnvConfig struct {
	Name               string `json:"name" yaml:"name"`
	OperatorEndpoint   string `json:"operator_endpoint" yaml:"operator_endpoint"`
	AWSAccessKeyID     string `json:"aws_access_key_id" yaml:"aws_access_key_id"`
	AWSSecretAccessKey string `json:"aws_secret_access_key" yaml:"aws_secret_access_key"`
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
								Required: true,
							},
						},
						{
							StructField: "OperatorEndpoint",
							StringValidation: &cr.StringValidation{
								Required:  true,
								Validator: cr.GetURLValidator(false, false),
							},
						},
						{
							StructField: "AWSAccessKeyID",
							StringValidation: &cr.StringValidation{
								Required: true,
							},
						},
						{
							StructField: "AWSSecretAccessKey",
							StringValidation: &cr.StringValidation{
								Required: true,
							},
						},
					},
				},
			},
		},
	},
}

func cliEnvPromptValidation(defaults *CLIEnvConfig) *cr.PromptValidation {
	if defaults == nil {
		defaults = &CLIEnvConfig{}
	}

	if defaults.AWSAccessKeyID == "" && os.Getenv("AWS_ACCESS_KEY_ID") != "" {
		defaults.AWSAccessKeyID = os.Getenv("AWS_ACCESS_KEY_ID")
	}
	if defaults.AWSSecretAccessKey == "" && os.Getenv("AWS_SECRET_ACCESS_KEY") != "" {
		defaults.AWSSecretAccessKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
	}
	if defaults.OperatorEndpoint == "" && os.Getenv("CORTEX_OPERATOR_ENDPOINT") != "" {
		defaults.OperatorEndpoint = os.Getenv("CORTEX_OPERATOR_ENDPOINT")
	}

	return &cr.PromptValidation{
		PromptItemValidations: []*cr.PromptItemValidation{
			{
				StructField: "OperatorEndpoint",
				PromptOpts: &prompt.Options{
					Prompt: "cortex operator endpoint",
				},
				StringValidation: &cr.StringValidation{
					Required:  true,
					Default:   defaults.OperatorEndpoint,
					Validator: cr.GetURLValidator(false, false),
				},
			},
			{
				StructField: "AWSAccessKeyID",
				PromptOpts: &prompt.Options{
					Prompt: "aws access key id",
				},
				StringValidation: &cr.StringValidation{
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
				StringValidation: &cr.StringValidation{
					Required: true,
					Default:  defaults.AWSSecretAccessKey,
				},
			},
		},
	}
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

// May return nil if not configured
func readCLIEnvConfig(environment string) (*CLIEnvConfig, error) {
	cliConfig, err := readCLIConfig()
	if err != nil {
		return nil, err
	}

	for _, cliEnvConfig := range cliConfig.Environments {
		if cliEnvConfig.Name == environment {
			return cliEnvConfig, nil
		}
	}

	return nil, nil
}

func isCLIEnvConfigured(environment string) (bool, error) {
	cliEnvConfig, err := readCLIEnvConfig(environment)
	if err != nil {
		return false, err
	}

	return cliEnvConfig != nil, nil
}

func readOrConfigureCLIEnv(environment string) (CLIEnvConfig, error) {
	currentCLIEnvConfig, err := readCLIEnvConfig(environment)
	if err != nil {
		return CLIEnvConfig{}, err
	}

	if currentCLIEnvConfig != nil {
		return *currentCLIEnvConfig, nil
	}

	return configureCLIEnv(environment)
}

func configureCLIEnv(environment string) (CLIEnvConfig, error) {
	prevCLIEnvConfig, err := readCLIEnvConfig(environment)
	if err != nil {
		return CLIEnvConfig{}, err
	}

	if environment != "default" {
		fmt.Println("environment: " + environment + "\n")
	}

	cliEnvConfig := CLIEnvConfig{
		Name: environment,
	}

	err = cr.ReadPrompt(&cliEnvConfig, cliEnvPromptValidation(prevCLIEnvConfig))
	if err != nil {
		return CLIEnvConfig{}, err
	}

	if err := addEnvToCLIConfig(cliEnvConfig); err != nil {
		return CLIEnvConfig{}, err
	}

	return cliEnvConfig, nil
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
		return CLIConfig{}, err
	}

	return cliConfig, nil
}

func (cliConfig CLIConfig) validate() error {
	envNames := strset.New()
	for _, cliEnvConfig := range cliConfig.Environments {
		if envNames.Has(cliEnvConfig.Name) {
			return errors.Wrap(ErrorDuplicateCLIEnvNames(cliEnvConfig.Name), _cliConfigPath, "environments")
		}
		envNames.Add(cliEnvConfig.Name)
	}
	return nil
}

func addEnvToCLIConfig(newCLIEnvConfig CLIEnvConfig) error {
	cliConfig, err := readCLIConfig()
	if err != nil {
		return err
	}

	replaced := false
	for i, prevCLIEnvConfig := range cliConfig.Environments {
		if prevCLIEnvConfig.Name == newCLIEnvConfig.Name {
			cliConfig.Environments[i] = &newCLIEnvConfig
			replaced = true
			break
		}
	}

	if !replaced {
		cliConfig.Environments = append(cliConfig.Environments, &newCLIEnvConfig)
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
