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

package cmd

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/cortexlabs/cortex/cli/cluster"
	"github.com/cortexlabs/cortex/cli/types/cliconfig"
	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/lib/print"
	"github.com/cortexlabs/cortex/pkg/lib/prompt"
	"github.com/cortexlabs/cortex/pkg/lib/slices"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/urls"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types"
	"github.com/cortexlabs/yaml"
)

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
			StringPtrValidation: &cr.StringPtrValidation{
				Required:          false,
				AllowExplicitNull: true,
			},
		},
		{
			StructField: "Environments",
			StructListValidation: &cr.StructListValidation{
				AllowExplicitNull: true,
				StructValidation: &cr.StructValidation{
					AllowExtraFields: true, // backwards compatibility with previous version cli.yaml that require aws creds
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
								Required: true,
								Validator: func(provider string) (string, error) {
									if slices.HasString(types.ProviderTypeStrings(), provider) {
										return provider, nil
									}
									if provider == "local" {
										return "", ErrorInvalidLegacyProvider(provider, _cliConfigPath)
									}
									return "", ErrorInvalidProvider(provider)
								},
							},
							Parser: func(str string) (interface{}, error) {
								return types.ProviderTypeFromString(str), nil
							},
						},
						{
							StructField: "OperatorEndpoint",
							StringValidation: &cr.StringValidation{
								Required:  true,
								Validator: cliconfig.CortexEndpointValidator,
							},
						},
					},
				},
			},
		},
	},
}

func getEnvFromFlag(envFlag string) (string, error) {
	if envFlag != "" {
		return envFlag, nil
	}

	defaultEnv, err := getDefaultEnv()
	if err != nil {
		return "", err
	}

	if defaultEnv != nil {
		return *defaultEnv, nil
	}

	envs, err := listConfiguredEnvs()
	if err != nil {
		return "", err
	}
	if len(envs) == 0 {
		return "", ErrorNoAvailableEnvironment()
	}

	return "", ErrorEnvironmentNotSet()
}

func promptForExistingEnvName(promptMsg string) string {
	configuredEnvNames, err := listConfiguredEnvNames()
	if err != nil {
		exit.Error(err)
	}

	fmt.Printf("your currently configured environments are: %s\n\n", strings.Join(configuredEnvNames, ", "))

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
					AllowedValues: configuredEnvNames,
				},
			},
		},
	})
	if err != nil {
		exit.Error(err)
	}

	return envNameContainer.EnvironmentName
}

func promptEnv(env *cliconfig.Environment, defaults cliconfig.Environment) error {
	if env.OperatorEndpoint == "" {
		fmt.Print("you can get your cortex operator endpoint using `cortex cluster info` or `cortex cluster-gcp info` if you already have a cortex cluster running, otherwise run `cortex cluster up` or `cortex cluster-gcp up` to create a cortex cluster\n\n")
	}

	validator := func(endpoint string) (string, error) {
		operatorURL, provider, err := validateOperatorEndpoint(endpoint)
		if err != nil {
			return "", err
		}

		env.Provider = provider
		return operatorURL, nil
	}

	for true {
		err := cr.ReadPrompt(env, &cr.PromptValidation{
			SkipNonEmptyFields: true,
			PromptItemValidations: []*cr.PromptItemValidation{
				{
					StructField: "Name",
					PromptOpts: &prompt.Options{
						Prompt: "name of environment to create or update",
					},
					StringValidation: &cr.StringValidation{
						Required: true,
					},
				},
				{
					StructField: "OperatorEndpoint",
					PromptOpts: &prompt.Options{
						Prompt: "cortex operator endpoint",
					},
					StringValidation: &cr.StringValidation{
						Required:  true,
						Default:   defaults.OperatorEndpoint,
						Validator: validator,
					},
				},
			},
		})
		if err != nil {
			return err
		}

		return nil
	}

	return nil
}

// Only validate this during prompt, not when reading from file
func validateOperatorEndpoint(endpoint string) (string, types.ProviderType, error) {
	url, err := cliconfig.CortexEndpointValidator(endpoint)
	if err != nil {
		return "", types.UnknownProviderType, err
	}

	parsedURL, err := urls.Parse(url)
	if err != nil {
		return "", types.UnknownProviderType, err
	}

	url = parsedURL.String()

	req, err := http.NewRequest("GET", urls.Join(url, "/verifycortex"), nil)
	if err != nil {
		return "", types.UnknownProviderType, errors.Wrap(err, "verifying operator endpoint", url)
	}

	client := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	response, err := client.Do(req)
	if err != nil {
		return "", types.UnknownProviderType, ErrorInvalidOperatorEndpoint(url)
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return "", types.UnknownProviderType, ErrorInvalidOperatorEndpoint(url)
	}

	bodyBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", types.UnknownProviderType, errors.Wrap(err, _errStrRead)
	}

	var verifyCortex schema.VerifyCortexResponse
	if err = json.Unmarshal(bodyBytes, &verifyCortex); err != nil {
		return "", types.UnknownProviderType, errors.Wrap(err, endpoint, string(bodyBytes))
	}

	return url, verifyCortex.Provider, nil
}

func getDefaultEnv() (*string, error) {
	cliConfig, err := readCLIConfig()
	if err != nil {
		return nil, err
	}

	if cliConfig.DefaultEnvironment != nil {
		return cliConfig.DefaultEnvironment, nil
	}

	if len(cliConfig.Environments) == 1 {
		defaultEnv := cliConfig.Environments[0].Name
		err := setDefaultEnv(defaultEnv)
		if err != nil {
			return nil, err
		}
		return &defaultEnv, nil
	}

	return nil, nil
}

func setDefaultEnv(envName string) error {
	cliConfig, err := readCLIConfig()
	if err != nil {
		return err
	}

	envExists, err := isEnvConfigured(envName)
	if err != nil {
		return err
	}
	if !envExists {
		return cliconfig.ErrorEnvironmentNotConfigured(envName)
	}

	cliConfig.DefaultEnvironment = &envName

	if err := writeCLIConfig(cliConfig); err != nil {
		return err
	}

	return nil
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

// Returns false if there is an error reading the CLI config
func isTelemetryEnabled() bool {
	enabled, err := readTelemetryConfig()
	if err != nil {
		return false
	}
	return enabled
}

// Will return nil if not configured
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

	return nil, nil
}

func ReadOrConfigureEnv(envName string) (cliconfig.Environment, error) {
	existingEnv, err := readEnv(envName)
	if err != nil {
		return cliconfig.Environment{}, err
	}

	if existingEnv != nil {
		return *existingEnv, nil
	}

	promptStr := fmt.Sprintf("the %s environment is not configured; do you already have a Cortex cluster running?", envName)
	yesMsg := fmt.Sprintf("please configure the %s environment to point to your running cluster:\n", envName)
	noMsg := "you can create a cluster on AWS or GCP by running the `cortex cluster up` or `cortex cluster-gcp up` command"
	prompt.YesOrExit(promptStr, yesMsg, noMsg)

	env, err := configureEnv(envName, cliconfig.Environment{})
	if err != nil {
		return cliconfig.Environment{}, err
	}

	return env, nil
}

func getEnvConfigDefaults(envName string) cliconfig.Environment {
	defaults := cliconfig.Environment{}

	prevEnv, err := readEnv(envName)
	if err == nil && prevEnv != nil {
		defaults = *prevEnv
	}

	if defaults.OperatorEndpoint == "" && os.Getenv("CORTEX_OPERATOR_ENDPOINT") != "" {
		defaults.OperatorEndpoint = os.Getenv("CORTEX_OPERATOR_ENDPOINT")
	}

	return defaults
}

// If envName is "", this will prompt for the environment name to configure
func configureEnv(envName string, fieldsToSkipPrompt cliconfig.Environment) (cliconfig.Environment, error) {
	env := cliconfig.Environment{
		Name:             envName,
		OperatorEndpoint: fieldsToSkipPrompt.OperatorEndpoint,
		Provider:         fieldsToSkipPrompt.Provider,
	}

	defaults := getEnvConfigDefaults(env.Name)

	err := promptEnv(&env, defaults)
	if err != nil {
		return cliconfig.Environment{}, err
	}

	if err := env.Validate(); err != nil {
		return cliconfig.Environment{}, err
	}

	if err := addEnvToCLIConfig(env, false); err != nil {
		return cliconfig.Environment{}, err
	}

	print.BoldFirstLine(fmt.Sprintf("configured %s environment", env.Name))

	return env, nil
}

func MustGetOperatorConfig(envName string) cluster.OperatorConfig {
	clientID := clientID()
	env, err := readEnv(envName)
	if err != nil {
		exit.Error(err)
	}

	if env == nil {
		exit.Error(ErrorEnvironmentNotFound(envName))
	}

	operatorConfig := cluster.OperatorConfig{
		Telemetry: isTelemetryEnabled(),
		ClientID:  clientID,
		EnvName:   env.Name,
		Provider:  env.Provider,
	}

	if env.OperatorEndpoint == "" {
		exit.Error(ErrorFieldNotFoundInEnvironment(cliconfig.OperatorEndpointKey, env.Name))
	}
	operatorConfig.OperatorEndpoint = env.OperatorEndpoint

	return operatorConfig
}

func listConfiguredEnvs() ([]*cliconfig.Environment, error) {
	cliConfig, err := readCLIConfig()
	if err != nil {
		return nil, err
	}

	return cliConfig.Environments, nil
}

func listConfiguredEnvNames() ([]string, error) {
	envList, err := listConfiguredEnvs()
	if err != nil {
		return nil, err
	}

	envNames := make([]string, len(envList))
	for i, env := range envList {
		envNames[i] = env.Name
	}

	return envNames, nil
}

func isEnvConfigured(envName string) (bool, error) {
	envList, err := listConfiguredEnvs()
	if err != nil {
		return false, err
	}

	for _, env := range envList {
		if env.Name == envName {
			return true, nil
		}
	}

	return false, nil
}

func addEnvToCLIConfig(newEnv cliconfig.Environment, setAsDefault bool) error {
	cliConfig, err := readCLIConfig()
	if err != nil {
		return errors.Wrap(err, "unable to configure cli environment")
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

	if setAsDefault {
		cliConfig.DefaultEnvironment = &newEnv.Name
	}

	if err := writeCLIConfig(cliConfig); err != nil {
		return errors.Wrap(err, "unable to configure cli environment")
	}

	return nil
}

func removeEnvFromCLIConfig(envName string) error {
	cliConfig, err := readCLIConfig()
	if err != nil {
		return err
	}

	prevDefault, err := getDefaultEnv()
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

	if !deleted {
		return cliconfig.ErrorEnvironmentNotConfigured(envName)
	}

	cliConfig.Environments = updatedEnvs

	if prevDefault != nil && envName == *prevDefault {
		cliConfig.DefaultEnvironment = nil
	}
	if len(cliConfig.Environments) == 1 {
		cliConfig.DefaultEnvironment = &cliConfig.Environments[0].Name
	}

	if err := writeCLIConfig(cliConfig); err != nil {
		return err
	}

	return nil
}

// returns the list of environment names, and whether any of them were the default
func getEnvNamesByOperatorEndpoint(operatorEndpoint string) ([]string, bool, error) {
	cliConfig, err := readCLIConfig()
	if err != nil {
		return nil, false, err
	}

	var envNames []string
	isDefaultEnv := false

	for _, env := range cliConfig.Environments {
		if env.OperatorEndpoint != "" && s.LastSplit(env.OperatorEndpoint, "//") == s.LastSplit(operatorEndpoint, "//") {
			envNames = append(envNames, env.Name)
			if cliConfig.DefaultEnvironment != nil && env.Name == *cliConfig.DefaultEnvironment {
				isDefaultEnv = true
			}
		}
	}

	return envNames, isDefaultEnv, nil
}

func readCLIConfig() (cliconfig.CLIConfig, error) {
	if !files.IsFile(_cliConfigPath) {
		cliConfig := cliconfig.CLIConfig{}

		if err := cliConfig.Validate(); err != nil {
			return cliconfig.CLIConfig{}, err // unexpected
		}

		// create file so that the file created by the manager container maintains current user permissions
		if err := writeCLIConfig(cliConfig); err != nil {
			return cliconfig.CLIConfig{}, errors.Wrap(err, "unable to save CLI configuration file")
		}

		return cliConfig, nil
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

	return nil
}

func listConfiguredEnvsForProvider(provider types.ProviderType) ([]*cliconfig.Environment, error) {
	cliConfig, err := readCLIConfig()
	if err != nil {
		return nil, err
	}

	var envs []*cliconfig.Environment
	for i := range cliConfig.Environments {
		if cliConfig.Environments[i].Provider == provider {
			envs = append(envs, cliConfig.Environments[i])
		}
	}

	return envs, nil
}

func listConfiguredEnvNamesForProvider(provider types.ProviderType) ([]string, error) {
	envList, err := listConfiguredEnvsForProvider(provider)
	if err != nil {
		return nil, err
	}

	envNames := make([]string, len(envList))
	for i, env := range envList {
		envNames[i] = env.Name
	}

	return envNames, nil
}
