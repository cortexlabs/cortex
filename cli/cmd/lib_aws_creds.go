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
	"fmt"
	"os"

	"github.com/cortexlabs/cortex/pkg/lib/aws"
	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/prompt"
)

type AWSCredentials struct {
	AWSAccessKeyID           string `json:"aws_access_key_id"`
	AWSSecretAccessKey       string `json:"aws_secret_access_key"`
	CortexAWSAccessKeyID     string `json:"cortex_aws_access_key_id"`
	CortexAWSSecretAccessKey string `json:"cortex_aws_secret_access_key"`
}

func newAWSClient(region string, awsCreds AWSCredentials) (*aws.Client, error) {
	awsClient, err := aws.NewFromCreds(region, awsCreds.AWSAccessKeyID, awsCreds.AWSSecretAccessKey)
	if err != nil {
		return nil, err
	}

	if _, _, err := awsClient.CheckCredentials(); err != nil {
		return nil, err
	}

	return awsClient, nil
}

var _awsCredentialsValidation = &cr.StructValidation{
	AllowExtraFields: true,
	StructFieldValidations: []*cr.StructFieldValidation{
		{
			StructField: "AWSAccessKeyID",
			StringValidation: &cr.StringValidation{
				AllowEmpty: true,
			},
		},
		{
			StructField: "AWSSecretAccessKey",
			StringValidation: &cr.StringValidation{
				AllowEmpty: true,
			},
		},
		{
			StructField: "CortexAWSAccessKeyID",
			StringValidation: &cr.StringValidation{
				AllowEmpty: true,
			},
		},
		{
			StructField: "CortexAWSSecretAccessKey",
			StringValidation: &cr.StringValidation{
				AllowEmpty: true,
			},
		},
	},
}

var _awsCredentialsPromptValidation = &cr.PromptValidation{
	PromptItemValidations: []*cr.PromptItemValidation{
		{
			StructField: "AWSAccessKeyID",
			PromptOpts: &prompt.Options{
				Prompt: "aws access key id",
			},
			StringValidation: &cr.StringValidation{
				Required: true,
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
			},
		},
	},
}

func readAWSCredsFromConfigFile(awsCreds *AWSCredentials, path string) error {
	errs := cr.ParseYAMLFile(awsCreds, _awsCredentialsValidation, path)
	if errors.HasError(errs) {
		return errors.FirstError(errs...)
	}

	return nil
}

// awsCreds is what was read from the cluster config YAML
func setInstallAWSCredentials(awsCreds *AWSCredentials) error {
	// First check env vars
	if os.Getenv("AWS_SESSION_TOKEN") != "" {
		fmt.Println("warning: credentials requiring aws session tokens are not supported")
	}

	if os.Getenv("AWS_ACCESS_KEY_ID") != "" && os.Getenv("AWS_SECRET_ACCESS_KEY") != "" {
		awsCreds.AWSAccessKeyID = os.Getenv("AWS_ACCESS_KEY_ID")
		awsCreds.AWSSecretAccessKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
		return nil
	}
	if os.Getenv("AWS_ACCESS_KEY_ID") == "" && os.Getenv("AWS_SECRET_ACCESS_KEY") != "" {
		err := errors.New("only $AWS_SECRET_ACCESS_KEY is set; please run `export AWS_ACCESS_KEY_ID=***`")
		return errors.SetUser(err)
	}
	if os.Getenv("AWS_ACCESS_KEY_ID") != "" && os.Getenv("AWS_SECRET_ACCESS_KEY") == "" {
		err := errors.New("only $AWS_ACCESS_KEY_ID is set; please run `export AWS_SECRET_ACCESS_KEY=***`")
		return errors.SetUser(err)
	}

	// Next check what was read from cluster config YAML
	if awsCreds.AWSAccessKeyID != "" && awsCreds.AWSSecretAccessKey != "" {
		return nil
	}
	if awsCreds.AWSAccessKeyID == "" && awsCreds.AWSSecretAccessKey != "" {
		return errors.New(fmt.Sprintf("only aws_secret_access_key is set in %s; please set aws_access_key_id as well", _flagClusterConfig))
	}
	if awsCreds.AWSAccessKeyID != "" && awsCreds.AWSSecretAccessKey == "" {
		return errors.New(fmt.Sprintf("only aws_access_key_id is set in %s; please set aws_secret_access_key as well", _flagClusterConfig))
	}

	// Next check AWS CLI config file
	accessKeyID, secretAccessKey, err := aws.GetCredentialsFromCLIConfigFile()
	if err == nil {
		awsCreds.AWSAccessKeyID = accessKeyID
		awsCreds.AWSSecretAccessKey = secretAccessKey
		return nil
	}

	// Next check Cortex CLI config file
	cliEnvConfig, err := readCLIEnvConfig(_flagEnv)
	if err != nil && cliEnvConfig != nil && cliEnvConfig.AWSAccessKeyID != "" && cliEnvConfig.AWSSecretAccessKey != "" {
		awsCreds.AWSAccessKeyID = cliEnvConfig.AWSAccessKeyID
		awsCreds.AWSSecretAccessKey = cliEnvConfig.AWSSecretAccessKey
		return nil
	}

	// Prompt
	err = cr.ReadPrompt(awsCreds, _awsCredentialsPromptValidation)
	if err != nil {
		return err
	}

	return nil
}

// awsCreds is what was read from the cluster config YAML, after being passed through setInstallAWSCredentials() (so those credentials should be set)
func setOperatorAWSCredentials(awsCreds *AWSCredentials) error {
	// First check env vars
	if os.Getenv("CORTEX_AWS_ACCESS_KEY_ID") != "" && os.Getenv("CORTEX_AWS_SECRET_ACCESS_KEY") != "" {
		awsCreds.CortexAWSAccessKeyID = os.Getenv("CORTEX_AWS_ACCESS_KEY_ID")
		awsCreds.CortexAWSSecretAccessKey = os.Getenv("CORTEX_AWS_SECRET_ACCESS_KEY")
		return nil
	}
	if os.Getenv("CORTEX_AWS_ACCESS_KEY_ID") == "" && os.Getenv("CORTEX_AWS_SECRET_ACCESS_KEY") != "" {
		err := errors.New("only $CORTEX_AWS_SECRET_ACCESS_KEY is set; please run `export CORTEX_AWS_ACCESS_KEY_ID=***`")
		return errors.SetUser(err)
	}
	if os.Getenv("CORTEX_AWS_ACCESS_KEY_ID") != "" && os.Getenv("CORTEX_AWS_SECRET_ACCESS_KEY") == "" {
		err := errors.New("only $CORTEX_AWS_ACCESS_KEY_ID is set; please run `export CORTEX_AWS_SECRET_ACCESS_KEY=***`")
		return errors.SetUser(err)
	}

	// Next check what was read from cluster config YAML
	if awsCreds.CortexAWSAccessKeyID != "" && awsCreds.CortexAWSSecretAccessKey != "" {
		return nil
	}
	if awsCreds.CortexAWSAccessKeyID == "" && awsCreds.CortexAWSSecretAccessKey != "" {
		err := errors.New(fmt.Sprintf("only cortex_aws_secret_access_key is set in %s; please set cortex_aws_access_key_id as well", _flagClusterConfig))
		return errors.SetUser(err)
	}
	if awsCreds.CortexAWSAccessKeyID != "" && awsCreds.CortexAWSSecretAccessKey == "" {
		err := errors.New(fmt.Sprintf("only cortex_aws_access_key_id is set in %s; please set cortex_aws_secret_access_key as well", _flagClusterConfig))
		return errors.SetUser(err)
	}

	// Default to primary AWS credentials
	awsCreds.CortexAWSAccessKeyID = awsCreds.AWSAccessKeyID
	awsCreds.CortexAWSSecretAccessKey = awsCreds.AWSSecretAccessKey
	return nil
}

func getAWSCredentials(userClusterConfigPath string) (AWSCredentials, error) {
	awsCreds := AWSCredentials{}

	if userClusterConfigPath != "" {
		readAWSCredsFromConfigFile(&awsCreds, userClusterConfigPath)
	}

	err := setInstallAWSCredentials(&awsCreds)
	if err != nil {
		return AWSCredentials{}, err
	}

	err = setOperatorAWSCredentials(&awsCreds)
	if err != nil {
		return AWSCredentials{}, err
	}

	return awsCreds, nil
}
