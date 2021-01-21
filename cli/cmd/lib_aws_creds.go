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
	"fmt"
	"os"
	"path/filepath"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	libjson "github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"

	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/prompt"
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
)

type AWSCredentials struct {
	AWSAccessKeyID            string `json:"aws_access_key_id"`
	AWSSecretAccessKey        string `json:"aws_secret_access_key"`
	ClusterAWSAccessKeyID     string `json:"cluster_aws_access_key_id"`
	ClusterAWSSecretAccessKey string `json:"cluster_aws_secret_access_key"`
}

func (awsCreds *AWSCredentials) MaskedString() string {
	return fmt.Sprintf("aws access key id ****%s and aws secret access key ****%s", s.LastNChars(awsCreds.AWSAccessKeyID, 4), s.LastNChars(awsCreds.AWSSecretAccessKey, 4))
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
		warningStr := fmt.Sprintf("warning: your IAM user%s does not have administrator access. This will likely prevent Cortex from installing correctly, so it is recommended to attach the AdministratorAccess policy to your IAM user (or to a group that your IAM user belongs to) via the AWS IAM console. If you'd like, you may provide separate credentials for your cluster to use after it's running (see https://docs.cortex.dev/v/%s/).\n\n", accessKeyMsg, consts.CortexVersionMinor)
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

// Deprecation: specifying aws creds in cluster configuration is no longer supported; returns an error if aws credentials were found in cluster configuration yaml
func detectAWSCredsInConfigFile(cmd, path string) error {
	credentialFieldKeys := strset.New("aws_access_key_id", "aws_secret_access_key", "cortex_aws_access_key_id", "cortex_aws_secret_access_key")
	fieldMap, err := cr.ReadYAMLFileStrMap(path)
	if err != nil {
		return nil
	}

	for key := range fieldMap {
		if credentialFieldKeys.Has(key) {
			return errors.Wrap(ErrorCredentialsInClusterConfig(cmd, path), path, key)
		}
	}

	return nil
}

func awsCredentialsForManagingCluster(accessConfig clusterconfig.AccessConfig, disallowPrompt bool) (AWSCredentials, error) {
	awsCredentials, err := awsCredentialsFromFlags()
	if err != nil {
		return AWSCredentials{}, err
	}

	if awsCredentials != nil {
		return *awsCredentials, nil
	}

	awsCredentials = awsCredentialsFromCache(accessConfig)
	if awsCredentials != nil {
		fmt.Println(fmt.Sprintf("using %s from cache (to use different credentials, specify the --aws-key and --aws-secret flags)\n", awsCredentials.MaskedString()))
		return *awsCredentials, nil
	}

	awsCredentials, err = awsCredentialsFromEnvVars()
	if err != nil {
		return AWSCredentials{}, err
	}

	if awsCredentials != nil {
		fmt.Println(fmt.Sprintf("using %s found in environment variables (to use different credentials, specify the --aws-key and --aws-secret flags)\n", awsCredentials.MaskedString()))
		return *awsCredentials, nil
	}

	awsCredentials = awsCredentialsFromSharedCreds()

	if awsCredentials != nil {
		fmt.Println(fmt.Sprintf("using %s from the \"default\" profile configured via `aws configure` (to use different credentials, specify the --aws-key and --aws-secret flags)\n", awsCredentials.MaskedString()))
		return *awsCredentials, nil
	}

	if disallowPrompt {
		return AWSCredentials{}, ErrorMissingAWSCredentials()
	}

	awsCredentials, err = awsCredentialsPrompt()
	if err != nil {
		return AWSCredentials{}, errors.Append(err, "\n\nit may be possible to avoid this error by specifying the --aws-key and --aws-secret flags")
	}

	return *awsCredentials, nil
}

// Returns true if the provided credentials match either the operator or the CLI credentials
func (awsCreds *AWSCredentials) ContainsCreds(accessKeyID string, secretAccessKey string) bool {
	if awsCreds.AWSAccessKeyID == accessKeyID && awsCreds.AWSSecretAccessKey == secretAccessKey {
		return true
	}
	if awsCreds.ClusterAWSAccessKeyID == accessKeyID && awsCreds.ClusterAWSSecretAccessKey == secretAccessKey {
		return true
	}
	return false
}

func awsCredentialsFromFlags() (*AWSCredentials, error) {
	credentials := AWSCredentials{}

	if _flagAWSAccessKeyID == "" && _flagAWSSecretAccessKey == "" {
		if _flagClusterAWSAccessKeyID != "" || _flagClusterAWSSecretAccessKey != "" {
			return nil, ErrorOnlyAWSClusterFlagSet()
		}
		return nil, nil
	}

	if _flagAWSSecretAccessKey == "" {
		return nil, ErrorOneAWSFlagSet("--aws-key", "--aws-secret")
	}
	if _flagAWSAccessKeyID == "" {
		return nil, ErrorOneAWSFlagSet("--aws-secret", "--aws-key")
	}

	credentials.AWSAccessKeyID = _flagAWSAccessKeyID
	credentials.AWSSecretAccessKey = _flagAWSSecretAccessKey

	if _flagClusterAWSAccessKeyID != "" || _flagClusterAWSSecretAccessKey != "" {
		if _flagClusterAWSAccessKeyID == "" {
			return nil, ErrorOneAWSFlagSet("--cluster-aws-key", "--cluster-aws-secret")
		}
		if _flagClusterAWSSecretAccessKey == "" {
			return nil, ErrorOneAWSFlagSet("--cluster-aws-secret", "--cluster-aws-key")
		}

		credentials.ClusterAWSAccessKeyID = _flagClusterAWSAccessKeyID
		credentials.ClusterAWSSecretAccessKey = _flagClusterAWSSecretAccessKey
	} else {
		credentials.ClusterAWSAccessKeyID = credentials.AWSAccessKeyID
		credentials.ClusterAWSSecretAccessKey = credentials.AWSSecretAccessKey
	}

	return &credentials, nil
}

func awsCredentialsFromEnvVars() (*AWSCredentials, error) {
	credentials := AWSCredentials{}

	if os.Getenv("AWS_SESSION_TOKEN") != "" {
		fmt.Println("warning: credentials requiring aws session tokens are not supported")
	}

	if os.Getenv("AWS_ACCESS_KEY_ID") == "" && os.Getenv("AWS_SECRET_ACCESS_KEY") == "" {
		if os.Getenv("CLUSTER_AWS_ACCESS_KEY_ID") != "" || os.Getenv("CLUSTER_AWS_SECRET_ACCESS_KEY") != "" {
			return nil, ErrorOnlyAWSClusterEnvVarSet()
		}
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

	if os.Getenv("CLUSTER_AWS_ACCESS_KEY_ID") != "" || os.Getenv("CLUSTER_AWS_SECRET_ACCESS_KEY") != "" {
		if os.Getenv("CLUSTER_AWS_ACCESS_KEY_ID") == "" && os.Getenv("CLUSTER_AWS_SECRET_ACCESS_KEY") != "" {
			return nil, ErrorOneAWSEnvVarSet("CLUSTER_AWS_SECRET_ACCESS_KEY", "CLUSTER_AWS_ACCESS_KEY_ID")
		}
		if os.Getenv("CLUSTER_AWS_ACCESS_KEY_ID") != "" && os.Getenv("CLUSTER_AWS_SECRET_ACCESS_KEY") == "" {
			return nil, ErrorOneAWSEnvVarSet("CLUSTER_AWS_ACCESS_KEY_ID", "CLUSTER_AWS_SECRET_ACCESS_KEY")
		}

		credentials.ClusterAWSAccessKeyID = os.Getenv("CLUSTER_AWS_ACCESS_KEY_ID")
		credentials.ClusterAWSSecretAccessKey = os.Getenv("CLUSTER_AWS_SECRET_ACCESS_KEY")
	} else {
		credentials.ClusterAWSAccessKeyID = credentials.AWSAccessKeyID
		credentials.ClusterAWSSecretAccessKey = credentials.AWSSecretAccessKey
	}

	return &credentials, nil
}

// Read from "default" profile from credentials specified by AWS_SHARED_CREDENTIALS_FILE (default path: ~/.aws/credentials).
// Returns nil if an error is encountered.
func awsCredentialsFromSharedCreds() *AWSCredentials {
	credentials := AWSCredentials{}
	accessKeyID, secretAccessKey, err := aws.GetCredentialsFromCLIConfigFile()
	if err != nil {
		return nil
	}

	credentials.AWSAccessKeyID = accessKeyID
	credentials.AWSSecretAccessKey = secretAccessKey
	credentials.ClusterAWSAccessKeyID = accessKeyID
	credentials.ClusterAWSSecretAccessKey = secretAccessKey
	return &credentials
}

func awsCredentialsPrompt() (*AWSCredentials, error) {
	credentials := AWSCredentials{}

	err := cr.ReadPrompt(&credentials, _awsCredentialsPromptValidation)
	if err != nil {
		return nil, err
	}

	credentials.ClusterAWSAccessKeyID = credentials.AWSAccessKeyID
	credentials.ClusterAWSSecretAccessKey = credentials.AWSSecretAccessKey

	return &credentials, nil
}

func credentialsCachePath(accessConfig clusterconfig.AccessConfig) string {
	return filepath.Join(_credentialsCacheDir, fmt.Sprintf("%s-%s.json", *accessConfig.Region, *accessConfig.ClusterName))
}

// Read AWS credentials from cache.
// Returns nil if not found or an error is encountered.
func awsCredentialsFromCache(accessConfig clusterconfig.AccessConfig) *AWSCredentials {

	credsPath := credentialsCachePath(accessConfig)

	if !files.IsFile(credsPath) {
		return nil
	}

	jsonBytes, err := files.ReadFileBytes(credsPath)
	if err != nil {
		return nil
	}

	credentials := AWSCredentials{}

	err = libjson.Unmarshal(jsonBytes, &credentials)
	if err != nil {
		return nil
	}

	return &credentials
}

func cacheAWSCredentials(awsCreds AWSCredentials, accessConfig clusterconfig.AccessConfig) error {
	jsonBytes, err := libjson.Marshal(awsCreds)
	if err != nil {
		return err
	}

	err = files.WriteFile(jsonBytes, credentialsCachePath(accessConfig))
	if err != nil {
		return err
	}

	return nil
}

func uncacheAWSCredentials(accessConfig clusterconfig.AccessConfig) error {
	return os.Remove(credentialsCachePath(accessConfig))
}
