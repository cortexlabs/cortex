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
	"io/ioutil"
	"os"
	"path/filepath"

	homedir "github.com/mitchellh/go-homedir"

	s "github.com/cortexlabs/cortex/pkg/api/strings"
	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	libjson "github.com/cortexlabs/cortex/pkg/lib/json"
)

var cachedCliConfig *CliConfig
var cachedCliConfigErrs []error
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

type CliConfig struct {
	CortexURL          string `json:"cortex_url"`
	AWSAccessKeyID     string `json:"aws_access_key_id"`
	AWSSecretAccessKey string `json:"aws_secret_access_key"`
}

func getPromptValidation(defaults *CliConfig) *cr.PromptValidation {
	return &cr.PromptValidation{
		PromptItemValidations: []*cr.PromptItemValidation{
			{
				StructField: "CortexURL",
				PromptOpts: &cr.PromptOptions{
					Prompt: "Enter Cortex operator endpoint",
				},
				StringValidation: cr.GetURLValidation(&cr.URLValidation{
					Required: true,
					Default:  defaults.CortexURL,
				}),
			},
			{
				StructField: "AWSAccessKeyID",
				PromptOpts: &cr.PromptOptions{
					Prompt: "Enter AWS Access Key ID",
				},
				StringValidation: &cr.StringValidation{
					Required: true,
					Default:  defaults.AWSAccessKeyID,
				},
			},
			{
				StructField: "AWSSecretAccessKey",
				PromptOpts: &cr.PromptOptions{
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
			StringValidation: cr.GetURLValidation(&cr.URLValidation{
				Required: true,
			}),
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

func readCliConfig() (*CliConfig, []error) {
	if cachedCliConfig != nil {
		return cachedCliConfig, cachedCliConfigErrs
	}

	configPath := configPath()
	cachedCliConfig = &CliConfig{}

	configBytes, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, []error{errors.Wrap(err, s.ErrReadFile(configPath))}
	}

	cliConfigData, err := cr.ReadJSONBytes(configBytes)
	if err != nil {
		cachedCliConfigErrs = []error{err}
		return cachedCliConfig, cachedCliConfigErrs
	}

	cachedCliConfigErrs = cr.Struct(cachedCliConfig, cliConfigData, fileValidation)
	return cachedCliConfig, errors.WrapMultiple(cachedCliConfigErrs, configPath)
}

func getValidCliConfig() *CliConfig {
	cliConfig, errs := readCliConfig()
	if errs != nil && len(errs) > 0 {
		cliConfig = configure()
	}
	return cliConfig
}

func getDefaults() *CliConfig {
	defaults, _ := readCliConfig()
	if defaults == nil {
		defaults = &CliConfig{}
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

func configure() *CliConfig {
	defaults := getDefaults()

	cachedCliConfig = &CliConfig{}
	fmt.Println("\nEnvironment: " + flagEnv + "\n")
	err := cr.ReadPrompt(cachedCliConfig, getPromptValidation(defaults))
	if err != nil {
		errors.Exit(err)
	}

	err = libjson.WriteJSON(cachedCliConfig, configPath())
	if err != nil {
		errors.Exit(err)
	}
	cachedCliConfigErrs = nil

	return cachedCliConfig
}
