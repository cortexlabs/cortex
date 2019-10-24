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

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/clusterconfig"
	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/prompt"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
)

type AWSCredentials struct {
	AWSAccessKeyID           string `json:"aws_access_key_id"`
	AWSSecretAccessKey       string `json:"aws_secret_access_key"`
	CortexAWSAccessKeyID     string `json:"cortex_aws_access_key_id"`
	CortexAWSSecretAccessKey string `json:"cortex_aws_secret_access_key"`
}

var awsCredentialsValidation = &cr.StructValidation{
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

var awsCredentialsPromptValidation = &cr.PromptValidation{
	PromptItemValidations: []*cr.PromptItemValidation{
		{
			StructField: "AWSAccessKeyID",
			PromptOpts: &prompt.Options{
				Prompt: "Enter AWS Access Key ID",
			},
			StringValidation: &cr.StringValidation{
				Required: true,
			},
		},
		{
			StructField: "AWSSecretAccessKey",
			PromptOpts: &prompt.Options{
				Prompt:      "Enter AWS Secret Access Key",
				MaskDefault: true,
				HideTyping:  true,
			},
			StringValidation: &cr.StringValidation{
				Required: true,
			},
		},
	},
}

// This does not set defaults for fields that are prompted from the user
func setClusterConfigFileDefaults(clusterConfig *clusterconfig.ClusterConfig) error {
	var emtpyMap interface{} = map[interface{}]interface{}{}
	errs := cr.Struct(clusterConfig, emtpyMap, clusterconfig.Validation)
	if errors.HasErrors(errs) {
		return errors.FirstError(errs...)
	}
	return nil
}

func readClusterConfigFile(clusterConfig *clusterconfig.ClusterConfig, awsCreds *AWSCredentials, path string) error {
	errs := cr.ParseYAMLFile(clusterConfig, clusterconfig.Validation, path)
	if errors.HasErrors(errs) {
		return errors.FirstError(errs...)
	}

	errs = cr.ParseYAMLFile(awsCreds, awsCredentialsValidation, path)
	if errors.HasErrors(errs) {
		return errors.FirstError(errs...)
	}

	return nil
}

// awsCreds is what was read from the cluster config YAML
func setInstallAWSCredentials(awsCreds *AWSCredentials) error {
	// First check env vars
	if os.Getenv("AWS_ACCESS_KEY_ID") != "" && os.Getenv("AWS_SECRET_ACCESS_KEY") != "" {
		awsCreds.AWSAccessKeyID = os.Getenv("AWS_ACCESS_KEY_ID")
		awsCreds.AWSSecretAccessKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
		return nil
	}
	if os.Getenv("AWS_ACCESS_KEY_ID") == "" && os.Getenv("AWS_SECRET_ACCESS_KEY") != "" {
		return errors.New("Only $AWS_SECRET_ACCESS_KEY is set; please run `export AWS_ACCESS_KEY_ID=***`")
	}
	if os.Getenv("AWS_ACCESS_KEY_ID") != "" && os.Getenv("AWS_SECRET_ACCESS_KEY") == "" {
		return errors.New("Only $AWS_ACCESS_KEY_ID is set; please run `export AWS_SECRET_ACCESS_KEY=***`")
	}

	// Next check what was read from cluster config YAML
	if awsCreds.AWSAccessKeyID != "" && awsCreds.AWSSecretAccessKey != "" {
		return nil
	}
	if awsCreds.AWSAccessKeyID == "" && awsCreds.AWSSecretAccessKey != "" {
		return errors.New(fmt.Sprintf("Only aws_secret_access_key is set in %s; please set aws_access_key_id as well", flagClusterConfig))
	}
	if awsCreds.AWSAccessKeyID != "" && awsCreds.AWSSecretAccessKey == "" {
		return errors.New(fmt.Sprintf("Only aws_access_key_id is set in %s; please set aws_secret_access_key as well", flagClusterConfig))
	}

	// Next check AWS CLI config file
	accessKeyID, secretAccessKey, err := aws.GetCredentialsFromCLIConfigFile()
	if err == nil {
		awsCreds.AWSAccessKeyID = accessKeyID
		awsCreds.AWSSecretAccessKey = secretAccessKey
		return nil
	}

	// Next check Cortex CLI config file
	cliConfig, errs := readCLIConfig()
	if !errors.HasErrors(errs) && cliConfig != nil && cliConfig.AWSAccessKeyID != "" && cliConfig.AWSSecretAccessKey != "" {
		awsCreds.AWSAccessKeyID = cliConfig.AWSAccessKeyID
		awsCreds.AWSSecretAccessKey = cliConfig.AWSSecretAccessKey
		return nil
	}

	// Prompt
	err = cr.ReadPrompt(awsCreds, awsCredentialsPromptValidation)
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
		return errors.New("Only $CORTEX_AWS_SECRET_ACCESS_KEY is set; please run `export CORTEX_AWS_ACCESS_KEY_ID=***`")
	}
	if os.Getenv("CORTEX_AWS_ACCESS_KEY_ID") != "" && os.Getenv("CORTEX_AWS_SECRET_ACCESS_KEY") == "" {
		return errors.New("Only $CORTEX_AWS_ACCESS_KEY_ID is set; please run `export CORTEX_AWS_SECRET_ACCESS_KEY=***`")
	}

	// Next check what was read from cluster config YAML
	if awsCreds.CortexAWSAccessKeyID != "" && awsCreds.CortexAWSSecretAccessKey != "" {
		return nil
	}
	if awsCreds.CortexAWSAccessKeyID == "" && awsCreds.CortexAWSSecretAccessKey != "" {
		return errors.New(fmt.Sprintf("Only cortex_aws_secret_access_key is set in %s; please set cortex_aws_access_key_id as well", flagClusterConfig))
	}
	if awsCreds.CortexAWSAccessKeyID != "" && awsCreds.CortexAWSSecretAccessKey == "" {
		return errors.New(fmt.Sprintf("Only cortex_aws_access_key_id is set in %s; please set cortex_aws_secret_access_key as well", flagClusterConfig))
	}

	// Default to primary AWS credentials
	awsCreds.CortexAWSAccessKeyID = awsCreds.AWSAccessKeyID
	awsCreds.CortexAWSSecretAccessKey = awsCreds.AWSSecretAccessKey
	return nil
}

// awsCreds is what was read from the cluster config YAML
func setAWSCredentials(awsCreds *AWSCredentials) error {
	err := setInstallAWSCredentials(awsCreds)
	if err != nil {
		return err
	}

	err = setOperatorAWSCredentials(awsCreds)
	if err != nil {
		return err
	}

	return nil
}

func confirmClusterConfig(clusterConfig *clusterconfig.ClusterConfig, awsCreds *AWSCredentials) {
	displayBucket := clusterConfig.Bucket
	if displayBucket == "" {
		displayBucket = "(autogenerate)"
	}

	fmt.Printf("instance type:     %s\n", *clusterConfig.InstanceType)
	fmt.Printf("min instances:     %d\n", *clusterConfig.MinInstances)
	fmt.Printf("max instances:     %d\n", *clusterConfig.MaxInstances)
	fmt.Printf("cluster name:      %s\n", clusterConfig.ClusterName)
	fmt.Printf("region:            %s\n", clusterConfig.Region)
	fmt.Printf("bucket:            %s\n", displayBucket)
	fmt.Printf("log group:         %s\n", clusterConfig.LogGroup)
	fmt.Printf("AWS access key ID: %s\n", s.MaskString(awsCreds.AWSAccessKeyID, 4))
	if awsCreds.CortexAWSAccessKeyID != awsCreds.AWSAccessKeyID {
		fmt.Printf("AWS access key ID: %s (cortex)\n", s.MaskString(awsCreds.CortexAWSAccessKeyID, 4))
	}
	fmt.Println()

	exitMessage := fmt.Sprintf("Cluster configuration can be modified via the cluster config file; see https://www.cortex.dev/v/%s/cluster-management/config", consts.CortexVersion)
	prompt.ForceYes("Is the configuration above correct?", exitMessage)
}

func getInstallClusterConfig() (*clusterconfig.ClusterConfig, *AWSCredentials, error) {
	clusterConfig := &clusterconfig.ClusterConfig{}
	awsCreds := &AWSCredentials{}

	if flagClusterConfig == "" {
		err := setClusterConfigFileDefaults(clusterConfig)
		if err != nil {
			return nil, nil, err
		}
	} else {
		err := readClusterConfigFile(clusterConfig, awsCreds, flagClusterConfig)
		if err != nil {
			return nil, nil, err
		}
	}

	err := cr.ReadPrompt(clusterConfig, clusterconfig.PromptValidation(true, nil))
	if err != nil {
		return nil, nil, err
	}

	err = setAWSCredentials(awsCreds)
	if err != nil {
		return nil, nil, err
	}

	confirmClusterConfig(clusterConfig, awsCreds)

	return clusterConfig, awsCreds, nil
}

func getUpdateClusterConfig() (*clusterconfig.ClusterConfig, *AWSCredentials, error) {
	clusterConfig := &clusterconfig.ClusterConfig{}
	awsCreds := &AWSCredentials{}

	if flagClusterConfig == "" {
		err := setClusterConfigFileDefaults(clusterConfig)
		if err != nil {
			return nil, nil, err
		}

		readClusterConfigFile(clusterConfig, awsCreds, cachedClusterConfigPath)

		err = cr.ReadPrompt(clusterConfig, clusterconfig.PromptValidation(false, clusterConfig))
		if err != nil {
			return nil, nil, err
		}
	} else {
		err := readClusterConfigFile(clusterConfig, awsCreds, flagClusterConfig)
		if err != nil {
			return nil, nil, err
		}

		err = cr.ReadPrompt(clusterConfig, clusterconfig.PromptValidation(true, nil))
		if err != nil {
			return nil, nil, err
		}
	}

	err := setAWSCredentials(awsCreds)
	if err != nil {
		return nil, nil, err
	}

	confirmClusterConfig(clusterConfig, awsCreds)

	return clusterConfig, awsCreds, nil
}

// This will only prompt for AWS credentials (if missing)
func getAccessClusterConfig() (*clusterconfig.ClusterConfig, *AWSCredentials, error) {
	clusterConfig := &clusterconfig.ClusterConfig{}
	awsCreds := &AWSCredentials{}

	if flagClusterConfig == "" {
		err := setClusterConfigFileDefaults(clusterConfig)
		if err != nil {
			return nil, nil, err
		}
		readClusterConfigFile(clusterConfig, awsCreds, cachedClusterConfigPath)
	} else {
		err := readClusterConfigFile(clusterConfig, awsCreds, flagClusterConfig)
		if err != nil {
			return nil, nil, err
		}
	}

	err := setAWSCredentials(awsCreds)
	if err != nil {
		return nil, nil, err
	}

	return clusterConfig, awsCreds, nil
}
