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
	"github.com/cortexlabs/cortex/pkg/lib/table"
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
				Prompt: "AWS Access Key ID",
			},
			StringValidation: &cr.StringValidation{
				Required: true,
			},
		},
		{
			StructField: "AWSSecretAccessKey",
			PromptOpts: &prompt.Options{
				Prompt:      "AWS Secret Access Key",
				MaskDefault: true,
				HideTyping:  true,
			},
			StringValidation: &cr.StringValidation{
				Required: true,
			},
		},
	},
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
	defaultCC, _ := clusterconfig.GetFileDefaults()

	var items []table.KV

	items = append(items, table.KV{K: "instance type", V: *clusterConfig.InstanceType})
	items = append(items, table.KV{K: "min instances", V: *clusterConfig.MinInstances})
	items = append(items, table.KV{K: "max instances", V: *clusterConfig.MaxInstances})
	items = append(items, table.KV{K: "cluster name", V: clusterConfig.ClusterName})
	items = append(items, table.KV{K: "region", V: clusterConfig.Region})
	items = append(items, table.KV{K: "bucket", V: clusterConfig.Bucket})

	if clusterConfig.LogGroup != defaultCC.LogGroup {
		items = append(items, table.KV{K: "log group", V: clusterConfig.LogGroup})
	}
	if clusterConfig.Telemetry != defaultCC.Telemetry {
		items = append(items, table.KV{K: "telemetry", V: clusterConfig.Telemetry})
	}

	items = append(items, table.KV{K: "AWS access key ID", V: s.MaskString(awsCreds.AWSAccessKeyID, 4)})
	if awsCreds.CortexAWSAccessKeyID != awsCreds.AWSAccessKeyID {
		items = append(items, table.KV{K: "AWS access key ID", V: s.MaskString(awsCreds.CortexAWSAccessKeyID, 4) + " (cortex)"})
	}

	if clusterConfig.ImagePredictorServe != defaultCC.ImagePredictorServe {
		items = append(items, table.KV{K: "image_predictor_serve", V: clusterConfig.ImagePredictorServe})
	}
	if clusterConfig.ImagePredictorServeGPU != defaultCC.ImagePredictorServeGPU {
		items = append(items, table.KV{K: "image_predictor_serve_gpu", V: clusterConfig.ImagePredictorServeGPU})
	}
	if clusterConfig.ImageTFServe != defaultCC.ImageTFServe {
		items = append(items, table.KV{K: "image_tf_serve", V: clusterConfig.ImageTFServe})
	}
	if clusterConfig.ImageTFServeGPU != defaultCC.ImageTFServeGPU {
		items = append(items, table.KV{K: "image_tf_serve_gpu", V: clusterConfig.ImageTFServeGPU})
	}
	if clusterConfig.ImageTFAPI != defaultCC.ImageTFAPI {
		items = append(items, table.KV{K: "image_tf_api", V: clusterConfig.ImageTFAPI})
	}
	if clusterConfig.ImageONNXServe != defaultCC.ImageONNXServe {
		items = append(items, table.KV{K: "image_onnx_serve", V: clusterConfig.ImageONNXServe})
	}
	if clusterConfig.ImageONNXServeGPU != defaultCC.ImageONNXServeGPU {
		items = append(items, table.KV{K: "image_onnx_serve_gpu", V: clusterConfig.ImageONNXServeGPU})
	}
	if clusterConfig.ImageOperator != defaultCC.ImageOperator {
		items = append(items, table.KV{K: "image_operator", V: clusterConfig.ImageOperator})
	}
	if clusterConfig.ImageManager != defaultCC.ImageManager {
		items = append(items, table.KV{K: "image_manager", V: clusterConfig.ImageManager})
	}
	if clusterConfig.ImageDownloader != defaultCC.ImageDownloader {
		items = append(items, table.KV{K: "image_downloader", V: clusterConfig.ImageDownloader})
	}
	if clusterConfig.ImageClusterAutoscaler != defaultCC.ImageClusterAutoscaler {
		items = append(items, table.KV{K: "image_cluster_autoscaler", V: clusterConfig.ImageClusterAutoscaler})
	}
	if clusterConfig.ImageMetricsServer != defaultCC.ImageMetricsServer {
		items = append(items, table.KV{K: "image_metrics_server", V: clusterConfig.ImageMetricsServer})
	}
	if clusterConfig.ImageNvidia != defaultCC.ImageNvidia {
		items = append(items, table.KV{K: "image_nvidia", V: clusterConfig.ImageNvidia})
	}
	if clusterConfig.ImageFluentd != defaultCC.ImageFluentd {
		items = append(items, table.KV{K: "image_fluentd", V: clusterConfig.ImageFluentd})
	}
	if clusterConfig.ImageStatsd != defaultCC.ImageStatsd {
		items = append(items, table.KV{K: "image_statsd", V: clusterConfig.ImageStatsd})
	}
	if clusterConfig.ImageIstioProxy != defaultCC.ImageIstioProxy {
		items = append(items, table.KV{K: "image_istio_proxy", V: clusterConfig.ImageIstioProxy})
	}
	if clusterConfig.ImageIstioPilot != defaultCC.ImageIstioPilot {
		items = append(items, table.KV{K: "image_istio_pilot", V: clusterConfig.ImageIstioPilot})
	}
	if clusterConfig.ImageIstioCitadel != defaultCC.ImageIstioCitadel {
		items = append(items, table.KV{K: "image_istio_citadel", V: clusterConfig.ImageIstioCitadel})
	}
	if clusterConfig.ImageIstioGalley != defaultCC.ImageIstioGalley {
		items = append(items, table.KV{K: "image_istio_galley", V: clusterConfig.ImageIstioGalley})
	}

	fmt.Println(table.AlignKeyValue(items, ":", 1) + "\n")

	exitMessage := fmt.Sprintf("Cluster configuration can be modified via the cluster config file; see https://www.cortex.dev/v/%s/cluster-management/config", consts.CortexVersion)
	prompt.YesOrExit("Is the configuration above correct?", exitMessage)
}

func getInstallClusterConfig() (*clusterconfig.ClusterConfig, *AWSCredentials, error) {
	clusterConfig := &clusterconfig.ClusterConfig{}
	awsCreds := &AWSCredentials{}

	if flagClusterConfig == "" {
		err := clusterconfig.SetFileDefaults(clusterConfig)
		if err != nil {
			return nil, nil, err
		}
	} else {
		err := readClusterConfigFile(clusterConfig, awsCreds, flagClusterConfig)
		if err != nil {
			return nil, nil, err
		}
	}

	err := cr.ReadPrompt(clusterConfig, clusterconfig.PromptValidation(true, true, nil))
	if err != nil {
		return nil, nil, err
	}

	err = setAWSCredentials(awsCreds)
	if err != nil {
		return nil, nil, err
	}

	err = clusterConfig.SetBucket(awsCreds.AWSAccessKeyID, awsCreds.AWSSecretAccessKey)
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
		err := clusterconfig.SetFileDefaults(clusterConfig)
		if err != nil {
			return nil, nil, err
		}

		readClusterConfigFile(clusterConfig, awsCreds, cachedClusterConfigPath)

		err = cr.ReadPrompt(clusterConfig, clusterconfig.PromptValidation(false, false, clusterConfig))
		if err != nil {
			return nil, nil, err
		}
	} else {
		err := readClusterConfigFile(clusterConfig, awsCreds, flagClusterConfig)
		if err != nil {
			return nil, nil, err
		}

		err = cr.ReadPrompt(clusterConfig, clusterconfig.PromptValidation(true, false, nil))
		if err != nil {
			return nil, nil, err
		}
	}

	err := setAWSCredentials(awsCreds)
	if err != nil {
		return nil, nil, err
	}

	err = clusterConfig.SetBucket(awsCreds.AWSAccessKeyID, awsCreds.AWSSecretAccessKey)
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
		err := clusterconfig.SetFileDefaults(clusterConfig)
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
