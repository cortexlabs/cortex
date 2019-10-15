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
	"path/filepath"

	homedir "github.com/mitchellh/go-homedir"

	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/lib/prompt"
)

var cachedCLIConfig *CLIConfig
var cachedCLIConfigErrs []error
var localDir string

func init() {
	dir, err := homedir.Dir()
	if err != nil {
		errors.Exit(err)
	}
	localDir = filepath.Join(dir, ".cortex")
	err = os.MkdirAll(localDir, os.ModePerm)
	if err != nil {
		errors.Exit(err)
	}
}

type CLIConfig struct {
	CortexURL          string `json:"cortex_url"`
	AWSAccessKeyID     string `json:"aws_access_key_id"`
	AWSSecretAccessKey string `json:"aws_secret_access_key"`
}

func getPromptValidation(defaults *CLIConfig) *cr.PromptValidation {
	return &cr.PromptValidation{
		PromptItemValidations: []*cr.PromptItemValidation{
			{
				StructField: "CortexURL",
				PromptOpts: &prompt.PromptOptions{
					Prompt: "Enter Cortex operator endpoint",
				},
				StringValidation: &cr.StringValidation{
					Required:  true,
					Default:   defaults.CortexURL,
					Validator: cr.GetURLValidator(false, false),
				},
			},
			{
				StructField: "AWSAccessKeyID",
				PromptOpts: &prompt.PromptOptions{
					Prompt: "Enter AWS Access Key ID",
				},
				StringValidation: &cr.StringValidation{
					Required: true,
					Default:  defaults.AWSAccessKeyID,
				},
			},
			{
				StructField: "AWSSecretAccessKey",
				PromptOpts: &prompt.PromptOptions{
					Prompt:      "Enter AWS Secret Access Key",
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

var fileValidation = &cr.StructValidation{
	ShortCircuit:     false,
	AllowExtraFields: true,
	StructFieldValidations: []*cr.StructFieldValidation{
		{
			Key:         "cortex_url",
			StructField: "CortexURL",
			StringValidation: &cr.StringValidation{
				Required:  true,
				Validator: cr.GetURLValidator(false, false),
			},
		},
		{
			Key:         "aws_access_key_id",
			StructField: "AWSAccessKeyID",
			StringValidation: &cr.StringValidation{
				Required: true,
			},
		},
		{
			Key:         "aws_secret_access_key",
			StructField: "AWSSecretAccessKey",
			StringValidation: &cr.StringValidation{
				Required: true,
			},
		},
	},
}

func configPath() string {
	return filepath.Join(localDir, flagEnv+".json")
}

func readCLIConfig() (*CLIConfig, []error) {
	if cachedCLIConfig != nil {
		return cachedCLIConfig, cachedCLIConfigErrs
	}

	configPath := configPath()
	cachedCLIConfig = &CLIConfig{}

	configBytes, err := files.ReadFileBytes(configPath)
	if err != nil {
		return nil, []error{err}
	}

	cliConfigData, err := cr.ReadJSONBytes(configBytes)
	if err != nil {
		cachedCLIConfigErrs = []error{err}
		return cachedCLIConfig, cachedCLIConfigErrs
	}

	cachedCLIConfigErrs = cr.Struct(cachedCLIConfig, cliConfigData, fileValidation)
	return cachedCLIConfig, errors.WrapMultiple(cachedCLIConfigErrs, configPath)
}

func getValidCLIConfig() *CLIConfig {
	cliConfig, errs := readCLIConfig()
	if len(errs) > 0 {
		cliConfig = configure()
	}
	return cliConfig
}

func getDefaults() *CLIConfig {
	defaults, _ := readCLIConfig()
	if defaults == nil {
		defaults = &CLIConfig{}
	}

	if defaults.AWSAccessKeyID == "" && os.Getenv("AWS_ACCESS_KEY_ID") != "" {
		defaults.AWSAccessKeyID = os.Getenv("AWS_ACCESS_KEY_ID")
	}
	if defaults.AWSSecretAccessKey == "" && os.Getenv("AWS_SECRET_ACCESS_KEY") != "" {
		defaults.AWSSecretAccessKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
	}
	if defaults.CortexURL == "" && os.Getenv("CORTEX_OPERATOR_ENDPOINT") != "" {
		defaults.CortexURL = os.Getenv("CORTEX_OPERATOR_ENDPOINT")
	}

	return defaults
}

func configure() *CLIConfig {
	defaults := getDefaults()
	cachedCLIConfig = &CLIConfig{}
	fmt.Println("\nEnvironment: " + flagEnv + "\n")
	err := cr.ReadPrompt(cachedCLIConfig, getPromptValidation(defaults))
	if err != nil {
		errors.Exit(err)
	}
	if err := json.WriteJSON(cachedCLIConfig, configPath()); err != nil {
		errors.Exit(err)
	}
	cachedCLIConfigErrs = nil

	return cachedCLIConfig
}
