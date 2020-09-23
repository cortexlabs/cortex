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
	"path/filepath"

	"github.com/cortexlabs/cortex/pkg/lib/aws"
	libjson "github.com/cortexlabs/cortex/pkg/lib/json"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"

	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/prompt"
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
)

type AWSCredentials struct {
	AWSAccessKeyID           string `json:"aws_access_key_id"`
	AWSSecretAccessKey       string `json:"aws_secret_access_key"`
	CortexAWSAccessKeyID     string `json:"cortex_aws_access_key_id"`
	CortexAWSSecretAccessKey string `json:"cortex_aws_secret_access_key"`
}

func (awsCredentials AWSCredentials) IsEmpty() bool {
	return len(awsCredentials.AWSAccessKeyID) == 0 && len(awsCredentials.AWSSecretAccessKey) == 0
}

func newAWSClient(region string, awsCreds AWSCredentials) (*aws.Client, error) {
	if err := clusterconfig.ValidateRegion(region); err != nil {
		return nil, err
	}

	awsClient, err := aws.NewFromCreds(region, awsCreds.AWSAccessKeyID, awsCreds.AWSSecretAccessKey)
	if err != nil {
		return nil, err
	}

	if _, _, err := awsClient.CheckCredentials(); err != nil {
		return nil, err
	}

	return awsClient, nil
}

func promptIfNotAdmin(awsClient *aws.Client, disallowPrompt bool) {
	accessKeyMsg := ""
	if accessKey := awsClient.AccessKeyID(); accessKey != nil {
		accessKeyMsg = fmt.Sprintf(" (with access key %s)", *accessKey)
	}

	if !awsClient.IsAdmin() {
		warningStr := fmt.Sprintf("warning: your IAM user%s does not have administrator access. This will likely prevent Cortex from installing correctly, so it is recommended to attach the AdministratorAccess policy to your IAM user (or to a group that your IAM user belongs to) via the AWS IAM console. If you'd like, you may provide separate credentials for your cluster to use after it's running (see https://docs.cortex.dev/miscellaneous/security for instructions).\n\n", accessKeyMsg)
		if disallowPrompt {
			fmt.Print(warningStr)
		} else {
			prompt.YesOrExit(warningStr+"are you sure you want to continue without administrator access?", "", "")
		}
	}
}

func warnIfNotAdmin(awsClient *aws.Client) {
	accessKeyMsg := ""
	if accessKey := awsClient.AccessKeyID(); accessKey != nil {
		accessKeyMsg = fmt.Sprintf(" (with access key %s)", *accessKey)
	}

	if !awsClient.IsAdmin() {
		fmt.Println(fmt.Sprintf("warning: your IAM user%s does not have administrator access. This may prevent this command from executing correctly, so it is recommended to attach the AdministratorAccess policy to your IAM user.", accessKeyMsg), "", "")
	}
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

func awsCredentialsForClusterUp(disallowPrompt bool) (AWSCredentials, error) {
	awsCredentials, err := awsCredentialsFromFlags()
	if err != nil {
		return AWSCredentials{}, err
	}

	if awsCredentials != nil {
		return *awsCredentials, nil
	}

	awsCredentials, err = awsCredentialsFromEnvVars()
	if err != nil {
		return AWSCredentials{}, err
	}

	if awsCredentials != nil {
		fmt.Println(fmt.Sprintf("using aws credentials AWS_ACCESS_KEY_ID=%s and AWS_SECRET_ACCESS_KEY=%s found in environment variables\n\nto use different credentials, specify the flags `--aws-key <AWS_ACCESS_KEY_ID> --aws-secret <AWS_SECRET_ACCESS_KEY>`\n", s.MaskString(awsCredentials.AWSAccessKeyID, 4), s.MaskString(awsCredentials.AWSSecretAccessKey, 4)))
		return *awsCredentials, nil
	}

	awsCredentials, err = awsCredentialsFromSharedCreds()
	if err != nil {
		return AWSCredentials{}, err
	}

	if awsCredentials != nil {
		fmt.Println(fmt.Sprintf("using aws credentials AWS_ACCESS_KEY_ID=%s and AWS_SECRET_ACCESS_KEY=%s from the default profile configured by the aws cli command: `aws configure`\n\nto use different credentials, specify the flags `--aws-key <AWS_ACCESS_KEY_ID> --aws-secret <AWS_SECRET_ACCESS_KEY>`\n", s.MaskString(awsCredentials.AWSAccessKeyID, 4), s.MaskString(awsCredentials.AWSSecretAccessKey, 4)))
		return *awsCredentials, nil
	}

	if !disallowPrompt {
		awsCredentials, err = awsCredentialsPrompt()
		if err != nil {
			return AWSCredentials{}, err
		}

		return *awsCredentials, nil
	}

	return AWSCredentials{}, ErrorMissingAWSCredentials()
}

func awsCredentialsForClusterCommands(accessConfig clusterconfig.AccessConfig, disallowPrompt bool) (AWSCredentials, error) {
	awsCredentials, err := awsCredentialsFromFlags()
	if err != nil {
		return AWSCredentials{}, err
	}

	if awsCredentials != nil {
		return *awsCredentials, nil
	}

	awsCredentials, err = getAWSCredentialsCortexCache(accessConfig)
	if err != nil {
		return AWSCredentials{}, err
	}

	if awsCredentials != nil {
		fmt.Println(fmt.Sprintf("using cached aws credentials AWS_ACCESS_KEY_ID=%s and AWS_SECRET_ACCESS_KEY=%s\n\nto use different credentials, specify the flags `--aws-key <AWS_ACCESS_KEY_ID> --aws-secret <AWS_SECRET_ACCESS_KEY>`\n", s.MaskString(awsCredentials.AWSAccessKeyID, 4), s.MaskString(awsCredentials.AWSSecretAccessKey, 4)))
		return *awsCredentials, nil
	}

	if !disallowPrompt {
		awsCredentials, err = awsCredentialsPrompt()
		if err != nil {
			return AWSCredentials{}, err
		}

		return *awsCredentials, nil
	}

	return AWSCredentials{}, ErrorMissingAWSCredentials()
}

// Returns true if the provided credentials match either the operator or the CLI credentials
func (awsCreds *AWSCredentials) ContainsCreds(accessKeyID string, secretAccessKey string) bool {
	if awsCreds.AWSAccessKeyID == accessKeyID && awsCreds.AWSSecretAccessKey == secretAccessKey {
		return true
	}
	if awsCreds.CortexAWSAccessKeyID == accessKeyID && awsCreds.CortexAWSSecretAccessKey == secretAccessKey {
		return true
	}
	return false
}

func awsCredentialsFromFlags() (*AWSCredentials, error) {
	credentials := AWSCredentials{}

	if len(_flagAWSAccessKeyID) == 0 && len(_flagAWSSecretAccessKey) == 0 {
		return nil, nil
	}

	if len(_flagAWSSecretAccessKey) == 0 {
		return nil, ErrorOneAWSFlagSet("--aws-key", "--aws-secret")
	}
	if len(_flagAWSAccessKeyID) == 0 {
		return nil, ErrorOneAWSFlagSet("--aws-secret", "--aws-key")
	}

	credentials.AWSAccessKeyID = _flagAWSAccessKeyID
	credentials.AWSSecretAccessKey = _flagAWSSecretAccessKey

	if len(_flagCortexAWSAccessKeyID) > 0 || len(_flagCortexAWSSecretAccessKey) > 0 {
		if len(_flagCortexAWSAccessKeyID) == 0 {
			return nil, ErrorOneAWSFlagSet("--aws-cortex-key", "--aws-cortex-secret")
		}
		if len(_flagCortexAWSSecretAccessKey) == 0 {
			return nil, ErrorOneAWSFlagSet("--aws-cortex-secret", "--aws-cortexkey")
		}

		credentials.CortexAWSAccessKeyID = _flagCortexAWSAccessKeyID
		credentials.CortexAWSSecretAccessKey = _flagCortexAWSAccessKeyID
	} else {
		credentials.CortexAWSAccessKeyID = credentials.AWSAccessKeyID
		credentials.CortexAWSSecretAccessKey = credentials.AWSSecretAccessKey
	}

	return &credentials, nil
}

func awsCredentialsFromEnvVars() (*AWSCredentials, error) {
	credentials := AWSCredentials{}

	if os.Getenv("AWS_SESSION_TOKEN") != "" {
		fmt.Println("warning: credentials requiring aws session tokens are not supported")
	}

	if os.Getenv("AWS_ACCESS_KEY_ID") == "" && os.Getenv("AWS_SECRET_ACCESS_KEY") == "" {
		return nil, nil
	}

	if os.Getenv("AWS_ACCESS_KEY_ID") == "" && os.Getenv("AWS_SECRET_ACCESS_KEY") != "" {
		return nil, ErrorOneAWSEnvVarSet("AWS_SECRET_ACCESS_KEY", "AWS_ACCESS_KEY_ID")
	}
	if os.Getenv("AWS_ACCESS_KEY_ID") != "" && os.Getenv("AWS_SECRET_ACCESS_KEY") == "" {
		return nil, ErrorOneAWSEnvVarSet("AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY")
	}

	credentials.AWSAccessKeyID = os.Getenv("AWS_ACCESS_KEY_ID")
	credentials.AWSSecretAccessKey = os.Getenv("AWS_SECRET_ACCESS_KEY")

	if os.Getenv("CORTEX_AWS_ACCESS_KEY_ID") != "" && os.Getenv("CORTEX_AWS_SECRET_ACCESS_KEY") != "" {
		credentials.CortexAWSAccessKeyID = os.Getenv("CORTEX_AWS_ACCESS_KEY_ID")
		credentials.CortexAWSSecretAccessKey = os.Getenv("CORTEX_AWS_SECRET_ACCESS_KEY")
	}
	if os.Getenv("CORTEX_AWS_ACCESS_KEY_ID") == "" && os.Getenv("CORTEX_AWS_SECRET_ACCESS_KEY") != "" {
		return nil, ErrorOneAWSEnvVarSet("CORTEX_AWS_SECRET_ACCESS_KEY", "CORTEX_AWS_ACCESS_KEY_ID")
	}
	if os.Getenv("CORTEX_AWS_ACCESS_KEY_ID") != "" && os.Getenv("CORTEX_AWS_SECRET_ACCESS_KEY") == "" {
		return nil, ErrorOneAWSEnvVarSet("CORTEX_AWS_ACCESS_KEY_ID", "CORTEX_AWS_SECRET_ACCESS_KEY")
	}

	credentials.CortexAWSAccessKeyID = credentials.AWSAccessKeyID
	credentials.CortexAWSSecretAccessKey = credentials.AWSSecretAccessKey

	return &credentials, nil
}

// Read from "default" profile from credentials specified by AWS_SHARED_CREDENTIALS_FILE (default path: ~/.aws/credentials)
func awsCredentialsFromSharedCreds() (*AWSCredentials, error) {
	credentials := AWSCredentials{}
	accessKeyID, secretAccessKey, err := aws.GetCredentialsFromCLIConfigFile()
	if err != nil {
		return nil, err
	}

	credentials.AWSAccessKeyID = accessKeyID
	credentials.AWSSecretAccessKey = secretAccessKey
	credentials.CortexAWSAccessKeyID = accessKeyID
	credentials.CortexAWSSecretAccessKey = secretAccessKey
	return &credentials, nil
}

func awsCredentialsPrompt() (*AWSCredentials, error) {
	credentials := AWSCredentials{}

	err := cr.ReadPrompt(&credentials, _awsCredentialsPromptValidation)
	if err != nil {
		return nil, err
	}

	credentials.CortexAWSAccessKeyID = credentials.AWSAccessKeyID
	credentials.CortexAWSSecretAccessKey = credentials.AWSSecretAccessKey

	return &credentials, nil
}

func credentialsCacheKey(accessConfig clusterconfig.AccessConfig) string {
	return filepath.Join(_credentialsCacheDir, fmt.Sprintf("%s-%s.json", *accessConfig.Region, *accessConfig.ClusterName))
}

func getAWSCredentialsCortexCache(accessConfig clusterconfig.AccessConfig) (*AWSCredentials, error) {
	credsPath := credentialsCacheKey(accessConfig)

	if err := files.CheckFile(credsPath); err != nil {
		return nil, nil
	}

	jsonBytes, err := files.ReadFileBytes(credsPath)
	if err != nil {
		return nil, err
	}

	credentials := AWSCredentials{}

	err = libjson.Unmarshal(jsonBytes, &credentials)
	if err != nil {
		return nil, err
	}

	return &credentials, nil
}

func cacheAWSCredentials(awsCreds AWSCredentials, accessConfig clusterconfig.AccessConfig) error {
	jsonBytes, err := libjson.Marshal(awsCreds)
	if err != nil {
		return err
	}

	err = files.WriteFile(jsonBytes, credentialsCacheKey(accessConfig))
	if err != nil {
		return err
	}

	return nil
}

func decacheAWSCredentials(accessConfig clusterconfig.AccessConfig) error {
	return os.Remove(credentialsCacheKey(accessConfig))
}
