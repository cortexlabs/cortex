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
	"strings"

	"github.com/cortexlabs/cortex/cli/cluster"
	"github.com/cortexlabs/cortex/cli/types/cliconfig"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/print"
	"github.com/cortexlabs/cortex/pkg/lib/prompt"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/urls"
	"github.com/cortexlabs/cortex/pkg/types"
	"github.com/cortexlabs/yaml"
)

var _cachedCLIConfig *cliconfig.CLIConfig

var _cliConfigValidation = &cr.StructValidation{
	TreatNullAsEmpty: true,
	StructFieldValidations: []*cr.StructFieldValidation{
		{
			StructField: "Telemetry",
			BoolPtrValidation: &cr.BoolPtrValidation{
				Required: false,
			},
		},
		{
			StructField: "DefaultEnvironment",
			StringValidation: &cr.StringValidation{
				Default:    "local",
				Required:   false,
				AllowEmpty: true, // will get set to "local" in validate() if empty
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

// This is a copy of _cliConfigValidation except for Provider; it can be deleted when removing the old CLI config conversion check
var _oldCLIConfigValidation = &cr.StructValidation{
	TreatNullAsEmpty: true,
	StructFieldValidations: []*cr.StructFieldValidation{
		{
			StructField: "Telemetry",
			BoolPtrValidation: &cr.BoolPtrValidation{
				Required: false,
			},
		},
		{
			StructField: "DefaultEnvironment",
			StringValidation: &cr.StringValidation{
				Default:    "local",
				Required:   false,
				AllowEmpty: true, // will get set to "local" in validate() if empty
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
								Required:   false,
								AllowEmpty: true,
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

// this checks for the old CLI configuration schema and updates it to the new one
// can be removed for 0.19 or 0.20 release
func convertOldCLIConfig() (cliconfig.CLIConfig, bool) {
	cliConfig := cliconfig.CLIConfig{}
	errs := cr.ParseYAMLFile(&cliConfig, _oldCLIConfigValidation, _cliConfigPath)
	if errors.HasError(errs) {
		return cliconfig.CLIConfig{}, false
	}

	if cliConfig.Telemetry != nil && *cliConfig.Telemetry == true {
		cliConfig.Telemetry = nil
	}

	var hasEnvNamedAWS bool
	for _, env := range cliConfig.Environments {
		if env.Name == types.AWSProviderType.String() {
			hasEnvNamedAWS = true
			break
		}
	}

	for _, env := range cliConfig.Environments {
		if env.Name == "default" && !hasEnvNamedAWS {
			env.Name = types.AWSProviderType.String()
		}

		// if provider is set, this is not an old CLI config
		if env.Provider != types.UnknownProviderType {
			return cliconfig.CLIConfig{}, false
		}

		env.Provider = types.AWSProviderType
	}

	if err := writeCLIConfig(cliConfig); err != nil {
		return cliconfig.CLIConfig{}, false
	}

	return cliConfig, true
}

func promptEnvName(promptMsg string, requireExistingEnv bool) string {
	configuredEnvNames, err := listConfiguredEnvs()
	if err != nil {
		exit.Error(err)
	}

	fmt.Printf("currently configured environments: %s\n\n", strings.Join(configuredEnvNames, ", "))

	var allowedValues []string
	if requireExistingEnv {
		allowedValues = configuredEnvNames
	}

	envNameContainer := &struct {
		EnvironmentName string
	}{}

	err = cr.ReadPrompt(envNameContainer, &cr.PromptValidation{
		PromptItemValidations: []*cr.PromptItemValidation{
			{
				StructField: "EnvironmentName",
				PromptOpts: &prompt.Options{
					Prompt: promptMsg,
				},
				StringValidation: &cr.StringValidation{
					Required:      true,
					AllowedValues: allowedValues,
				},
			},
		},
	})
	if err != nil {
		if err != nil {
			exit.Error(err)
		}
	}

	return envNameContainer.EnvironmentName
}

func promptProvider(env *cliconfig.Environment, envName string, defaults cliconfig.Environment) error {
	defaultProviderStr := ""
	if defaults.Provider != types.UnknownProviderType {
		defaultProviderStr = defaults.Provider.String()
	}

	return cr.ReadPrompt(env, &cr.PromptValidation{
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
	})
}

func promptLocalEnv(env *cliconfig.Environment, defaults cliconfig.Environment) error {
	accessKeyIDPrompt := "aws access key id"
	if defaults.AWSAccessKeyID == nil {
		accessKeyIDPrompt += " [press ENTER to skip]"
	}

	err := cr.ReadPrompt(env, &cr.PromptValidation{
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
		},
	})
	if err != nil {
		return err
	}

	// Don't prompt for secret access key if access key ID was not provided
	if env.AWSAccessKeyID == nil {
		env.AWSSecretAccessKey = nil
		return nil
	}

	return cr.ReadPrompt(env, &cr.PromptValidation{
		SkipNonEmptyFields: true,
		PromptItemValidations: []*cr.PromptItemValidation{
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
			{
				StructField: "AWSRegion",
				PromptOpts: &prompt.Options{
					Prompt:      "aws region",
					MaskDefault: true,
					HideTyping:  true,
				},
				StringPtrValidation: &cr.StringPtrValidation{
					Required:      true,
					Default:       pointer.String("us-east-1"),
					AllowedValues: aws.S3Regions.Slice(),
				},
			},
		},
	})
}

func promptAWSEnv(env *cliconfig.Environment, defaults cliconfig.Environment) error {
	return cr.ReadPrompt(env, &cr.PromptValidation{
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
	})
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

	if cliConfig.Telemetry != nil && *cliConfig.Telemetry == false {
		return false, nil
	}

	return true, nil
}

// Returns "local" if unable to read user's default value
func getDefaultEnv(cmdType commandType) string {
	defaultEnv := types.LocalProviderType.String()

	if cliConfig, err := readCLIConfig(); err == nil {
		defaultEnv = cliConfig.DefaultEnvironment
	}

	if cmdType == _clusterCommandType && defaultEnv == types.LocalProviderType.String() {
		defaultEnv = types.AWSProviderType.String()
	}

	return defaultEnv
}

func setDefaultEnv(envName string) error {
	cliConfig, err := readCLIConfig()
	if err != nil {
		return err
	}

	cliConfig.DefaultEnvironment = envName

	if err := writeCLIConfig(cliConfig); err != nil {
		return err
	}

	return nil
}

// Returns false if there is an error reading the CLI config
func isTelemetryEnabled() bool {
	enabled, err := readTelemetryConfig()
	if err != nil {
		return false
	}
	return enabled
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

// If envName is "", this will prompt for the environment name to configure
func configureEnv(envName string, fieldsToSkipPrompt cliconfig.Environment) (cliconfig.Environment, error) {
	if envName == "" {
		envName = promptEnvName("name of environment to update or create", false)
	} else {
		fmt.Println("environment: " + envName + "\n")
	}

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

	err := promptProvider(&env, envName, defaults)
	if err != nil {
		return cliconfig.Environment{}, err
	}

	switch env.Provider {
	case types.LocalProviderType:
		err = promptLocalEnv(&env, defaults)
	case types.AWSProviderType:
		err = promptAWSEnv(&env, defaults)
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

	print.BoldFirstLine(fmt.Sprintf("✓ configured %s environment", envName))

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

func listConfiguredEnvs() ([]string, error) {
	cliConfig, err := readCLIConfig()
	if err != nil {
		return nil, err
	}

	envNames := make([]string, len(cliConfig.Environments))
	for i, env := range cliConfig.Environments {
		envNames[i] = env.Name
	}

	return envNames, nil
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

	if err := writeCLIConfig(cliConfig); err != nil {
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

	if err := writeCLIConfig(cliConfig); err != nil {
		return err
	}

	return nil
}

func readCLIConfig() (cliconfig.CLIConfig, error) {
	if _cachedCLIConfig != nil {
		return *_cachedCLIConfig, nil
	}

	if !files.IsFile(_cliConfigPath) {
		cliConfig := cliconfig.CLIConfig{
			DefaultEnvironment: types.LocalProviderType.String(),
			Environments: []*cliconfig.Environment{
				{
					Name:     types.LocalProviderType.String(),
					Provider: types.LocalProviderType,
				},
			},
		}

		if err := cliConfig.Validate(); err != nil {
			return cliconfig.CLIConfig{}, err // unexpected
		}

		// create file so that the file created by the manager container maintains current user permissions
		if err := writeCLIConfig(cliConfig); err != nil {
			return cliconfig.CLIConfig{}, errors.Wrap(err, "unable to save CLI configuration file")
		}

		_cachedCLIConfig = &cliConfig
		return cliConfig, nil
	}

	cliConfig := cliconfig.CLIConfig{}
	errs := cr.ParseYAMLFile(&cliConfig, _cliConfigValidation, _cliConfigPath)
	if errors.HasError(errs) {
		var succeeded bool
		cliConfig, succeeded = convertOldCLIConfig()
		if !succeeded {
			return cliconfig.CLIConfig{}, errors.FirstError(errs...)
		}
	}

	if err := cliConfig.Validate(); err != nil {
		return cliconfig.CLIConfig{}, errors.Wrap(err, _cliConfigPath)
	}

	_cachedCLIConfig = &cliConfig
	return cliConfig, nil
}

func writeCLIConfig(cliConfig cliconfig.CLIConfig) error {
	if err := cliConfig.Validate(); err != nil {
		return err
	}

	cliConfigBytes, err := yaml.Marshal(cliConfig)
	if err != nil {
		return errors.WithStack(err)
	}
	if err := files.WriteFile(cliConfigBytes, _cliConfigPath); err != nil {
		return err
	}

	_cachedCLIConfig = &cliConfig
	return nil
}
