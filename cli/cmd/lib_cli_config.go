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
	"net/http"
	"os"
	"path/filepath"

	homedir "github.com/mitchellh/go-homedir"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/cloud"
	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/operator/api/schema"
)

const authHeaderTemplate = "CortexAuth %s"

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
	CortexURL     string             `json:"cortex_url"`
	CloudProvider cloud.ProviderType `json:"cloud_provider_type"`
	AWS           *AWSConfig         `json:"aws"`
	Local         *LocalConfig       `json:"local"`
}

func (c *CliConfig) authHeader() string {
	switch c.CloudProvider {
	case cloud.AWSProviderType:
		// | is not a valid character in access key or secret key
		return fmt.Sprintf(authHeaderTemplate, c.AWS.AccessKeyID+"|"+c.AWS.SecretAccessKey)
	case cloud.LocalProviderType:
		return fmt.Sprintf(authHeaderTemplate, "local")
	default:
		errors.Exit(ErrorUnrecognizedCloudProvider(c.CloudProvider.String()))
	}
	return ""
}

func cliConfigValidation(defaults *CliConfig) *cr.PromptValidation {
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
		},
	}
}

type LocalConfig struct {
	Mount string `json:"mount"`
}

func localPromptValidation(defaults *LocalConfig) *cr.PromptValidation {
	return &cr.PromptValidation{
		PromptItemValidations: []*cr.PromptItemValidation{
			{
				StructField: "Mount",
				PromptOpts: &cr.PromptOptions{
					Prompt: "Enter directory for Cortex mount",
				},
				StringValidation: &cr.StringValidation{
					Required: true,
					Default:  defaults.Mount,
				},
			},
		},
	}
}

type AWSConfig struct {
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
}

func awsPromptValidation(defaults *AWSConfig) *cr.PromptValidation {
	return &cr.PromptValidation{
		PromptItemValidations: []*cr.PromptItemValidation{
			{
				StructField: "AccessKeyID",
				PromptOpts: &cr.PromptOptions{
					Prompt: "Enter AWS Access Key ID",
				},
				StringValidation: &cr.StringValidation{
					Required: true,
					Default:  defaults.AccessKeyID,
				},
			},
			{
				StructField: "SecretAccessKey",
				PromptOpts: &cr.PromptOptions{
					Prompt:      "Enter AWS Secret Access Key",
					MaskDefault: true,
					HideTyping:  true,
				},
				StringValidation: &cr.StringValidation{
					Required: true,
					Default:  defaults.SecretAccessKey,
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
			StructField: "CortexURL",
			StringValidation: cr.GetURLValidation(&cr.URLValidation{
				Required: true,
			}),
		},
		{
			StructField: "CloudProvider",
			StringValidation: &cr.StringValidation{
				Required:      true,
				AllowedValues: cloud.ProviderTypeStrings(),
			},
			Parser: func(str string) (interface{}, error) {
				return cloud.ProviderTypeFromString(str), nil
			},
		},
		{
			StructField: "AWS",
			StructValidation: &cr.StructValidation{
				StructFieldValidations: []*cr.StructFieldValidation{
					{
						StructField: "AccessKeyID",
						StringValidation: &cr.StringValidation{
							Required: true,
						},
					},
					{
						StructField: "SecretAccessKey",
						StringValidation: &cr.StringValidation{
							Required: true,
						},
					},
				},
				Required:          false,
				AllowExplicitNull: true,
			},
		},
		{
			StructField: "Local",
			StructValidation: &cr.StructValidation{
				StructFieldValidations: []*cr.StructFieldValidation{
					{
						StructField: "Mount",
						StringValidation: &cr.StringValidation{
							Required: true,
						},
					},
				},
				Required:          false,
				AllowExplicitNull: true,
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

	configBytes, err := files.ReadFileBytes(configPath)
	if err != nil {
		return nil, []error{err}
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

func getCliConfigDefaults() *CliConfig {
	defaults, _ := readCliConfig()
	if defaults == nil {
		defaults = &CliConfig{}
	}

	if defaults.CortexURL == "" && os.Getenv("CORTEX_OPERATOR_ENDPOINT") != "" {
		defaults.CortexURL = os.Getenv("CORTEX_OPERATOR_ENDPOINT")
	}

	return defaults
}

func getAWSConfigDefaults(defaults *AWSConfig) *AWSConfig {
	if defaults == nil {
		defaults = &AWSConfig{}
	}
	if defaults.AccessKeyID == "" && os.Getenv("AWS_ACCESS_KEY_ID") != "" {
		defaults.AccessKeyID = os.Getenv("AWS_ACCESS_KEY_ID")
	}
	if defaults.SecretAccessKey == "" && os.Getenv("AWS_SECRET_ACCESS_KEY") != "" {
		defaults.SecretAccessKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
	}

	return defaults
}

func getLocalConfigDefaults(defaults *LocalConfig) *LocalConfig {
	if defaults == nil {
		defaults = &LocalConfig{}
	}
	if defaults.Mount == "" && os.Getenv("CORTEX_OPERATOR_LOCAL_MOUNT") != "" {
		defaults.Mount = os.Getenv("CORTEX_OPERATOR_LOCAL_MOUNT")
	}
	if defaults.Mount == "" {
		homeDir, err := homedir.Expand("~/.cortex/mount")
		if err != nil {
			errors.Exit(err)
		}
		defaults.Mount = homeDir
	}
	return defaults
}

func configure() *CliConfig {
	defaults := getCliConfigDefaults()

	cachedCliConfig = &CliConfig{}
	fmt.Println("\nEnvironment: " + flagEnv + "\n")
	err := cr.ReadPrompt(cachedCliConfig, cliConfigValidation(defaults))
	if err != nil {
		errors.Exit(err)
	}

	req, err := http.NewRequest("GET", cachedCliConfig.CortexURL+"/info", nil)
	req.Header.Set("CortexAPIVersion", consts.CortexVersion)
	response, err := httpClient.Do(req)
	if err != nil {
		errors.Exit(ErrorFailedToConnect(cachedCliConfig.CortexURL))
	}

	resBytes, err := handleResponse(response)
	if err != nil {
		errors.Exit(err)
	}

	var initRes schema.InfoResponse
	if err = json.Unmarshal(resBytes, &initRes); err != nil {
		errors.Exit(err)
	}

	switch initRes.CloudProvider {
	case cloud.AWSProviderType:
		cachedCliConfig.AWS = &AWSConfig{}
		err := cr.ReadPrompt(cachedCliConfig.AWS, awsPromptValidation(getAWSConfigDefaults(defaults.AWS)))
		if err != nil {
			errors.Exit(err)
		}
		// cachedCliConfig.AWS = awsConfig
	case cloud.LocalProviderType:
		cachedCliConfig.Local = &LocalConfig{}
		err := cr.ReadPrompt(cachedCliConfig.Local, localPromptValidation(getLocalConfigDefaults(defaults.Local)))
		if err != nil {
			errors.Exit(err)
		}
	default:
		errors.Exit(ErrorUnrecognizedCloudProvider(initRes.CloudProvider.String()))
	}

	cachedCliConfig.CloudProvider = initRes.CloudProvider

	err = json.WriteJSON(cachedCliConfig, configPath())
	if err != nil {
		errors.Exit(err)
	}
	cachedCliConfigErrs = nil

	return cachedCliConfig
}
