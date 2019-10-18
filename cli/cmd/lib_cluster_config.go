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
	"strings"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/prompt"
	"github.com/cortexlabs/cortex/pkg/operator/api/userconfig"
)

type ClusterConfig struct {
	AWSAccessKeyID           string  `json:"aws_access_key_id"`
	AWSSecretAccessKey       string  `json:"aws_secret_access_key"`
	CortexAWSAccessKeyID     string  `json:"cortex_aws_access_key_id"`
	CortexAWSSecretAccessKey string  `json:"cortex_aws_secret_access_key"`
	InstanceType             *string `json:"instance_type"`
	MinInstances             *int64  `json:"min_instances"`
	MaxInstances             *int64  `json:"max_instances"`
	ClusterName              string  `json:"cluster_name"`
	Region                   string  `json:"region"`
	Bucket                   string  `json:"bucket"`
	LogGroup                 string  `json:"log_group"`
	Telemetry                bool    `json:"telemetry"`
	ImageManager             string  `json:"image_manager"`
	ImageFluentd             string  `json:"image_fluentd"`
	ImageStatsd              string  `json:"image_statsd"`
	ImageOperator            string  `json:"image_operator"`
	ImageTFServe             string  `json:"image_tf_serve"`
	ImageTFAPI               string  `json:"image_tf_api"`
	ImageTFServeGPU          string  `json:"image_tf_serve_gpu"`
	ImageOnnxServe           string  `json:"image_onnx_serve"`
	ImageOnnxServeGPU        string  `json:"image_onnx_serve_gpu"`
	ImageClusterAutoscaler   string  `json:"image_cluster_autoscaler"`
	ImageNvidia              string  `json:"image_nvidia"`
	ImageMetricsServer       string  `json:"image_metrics_server"`
	ImageIstioCitadel        string  `json:"image_istio_citadel"`
	ImageIstioGalley         string  `json:"image_istio_galley"`
	ImageIstioPilot          string  `json:"image_istio_pilot"`
	ImageIstioProxy          string  `json:"image_istio_proxy"`
	ImageDownloader          string  `json:"image_downloader"`
}

var clusterConfigValidation = &cr.StructValidation{
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
		{
			StructField: "InstanceType",
			StringPtrValidation: &cr.StringPtrValidation{
				Validator: validateInstanceType,
			},
		},
		{
			StructField: "MinInstances",
			Int64PtrValidation: &cr.Int64PtrValidation{
				GreaterThan: pointer.Int64(0),
			},
		},
		{
			StructField: "MaxInstances",
			Int64PtrValidation: &cr.Int64PtrValidation{
				GreaterThan: pointer.Int64(0),
			},
		},
		{
			StructField: "ClusterName",
			StringValidation: &cr.StringValidation{
				Default: "cortex:" + consts.CortexVersion,
			},
		},
		{
			StructField: "Region",
			StringValidation: &cr.StringValidation{
				Default: "us-west-2",
			},
		},
		{
			StructField: "Bucket",
			StringValidation: &cr.StringValidation{
				Default:    "",
				AllowEmpty: true,
			},
		},
		{
			StructField: "LogGroup",
			StringValidation: &cr.StringValidation{
				Default: "cortex:" + consts.CortexVersion,
			},
		},
		{
			StructField: "Telemetry",
			BoolValidation: &cr.BoolValidation{
				Default: true,
			},
		},
		{
			StructField: "ImageManager",
			StringValidation: &cr.StringValidation{
				Default: "cortexlabs/manager:" + consts.CortexVersion,
			},
		},
		{
			StructField: "ImageFluentd",
			StringValidation: &cr.StringValidation{
				Default: "cortexlabs/fluentd:" + consts.CortexVersion,
			},
		},
		{
			StructField: "ImageStatsd",
			StringValidation: &cr.StringValidation{
				Default: "cortexlabs/statsd:" + consts.CortexVersion,
			},
		},
		{
			StructField: "ImageOperator",
			StringValidation: &cr.StringValidation{
				Default: "cortexlabs/operator:" + consts.CortexVersion,
			},
		},
		{
			StructField: "ImageTFServe",
			StringValidation: &cr.StringValidation{
				Default: "cortexlabs/tf-serve:" + consts.CortexVersion,
			},
		},
		{
			StructField: "ImageTFAPI",
			StringValidation: &cr.StringValidation{
				Default: "cortexlabs/tf-api:" + consts.CortexVersion,
			},
		},
		{
			StructField: "ImageTFServeGPU",
			StringValidation: &cr.StringValidation{
				Default: "cortexlabs/tf-serve-gpu:" + consts.CortexVersion,
			},
		},
		{
			StructField: "ImageOnnxServe",
			StringValidation: &cr.StringValidation{
				Default: "cortexlabs/onnx-serve:" + consts.CortexVersion,
			},
		},
		{
			StructField: "ImageOnnxServeGPU",
			StringValidation: &cr.StringValidation{
				Default: "cortexlabs/onnx-serve-gpu:" + consts.CortexVersion,
			},
		},
		{
			StructField: "ImageClusterAutoscaler",
			StringValidation: &cr.StringValidation{
				Default: "cortexlabs/cluster-autoscaler:" + consts.CortexVersion,
			},
		},
		{
			StructField: "ImageNvidia",
			StringValidation: &cr.StringValidation{
				Default: "cortexlabs/nvidia:" + consts.CortexVersion,
			},
		},
		{
			StructField: "ImageMetricsServer",
			StringValidation: &cr.StringValidation{
				Default: "cortexlabs/metrics-server:" + consts.CortexVersion,
			},
		},
		{
			StructField: "ImageIstioCitadel",
			StringValidation: &cr.StringValidation{
				Default: "cortexlabs/istio-citadel:" + consts.CortexVersion,
			},
		},
		{
			StructField: "ImageIstioGalley",
			StringValidation: &cr.StringValidation{
				Default: "cortexlabs/istio-galley:" + consts.CortexVersion,
			},
		},
		{
			StructField: "ImageIstioPilot",
			StringValidation: &cr.StringValidation{
				Default: "cortexlabs/istio-pilot:" + consts.CortexVersion,
			},
		},
		{
			StructField: "ImageIstioProxy",
			StringValidation: &cr.StringValidation{
				Default: "cortexlabs/istio-proxy:" + consts.CortexVersion,
			},
		},
		{
			StructField: "ImageDownloader",
			StringValidation: &cr.StringValidation{
				Default: "cortexlabs/downloader:" + consts.CortexVersion,
			},
		},
	},
}

var clusterConfigPrompt = &cr.PromptValidation{
	SkipPopulatedFields: true,
	PromptItemValidations: []*cr.PromptItemValidation{
		{
			StructField: "InstanceType",
			PromptOpts: &prompt.PromptOptions{
				Prompt: "AWS instance type",
			},
			StringPtrValidation: &cr.StringPtrValidation{
				Required:  true,
				Default:   pointer.String("m5.large"),
				Validator: validateInstanceType,
			},
		},
		{
			StructField: "MinInstances",
			PromptOpts: &prompt.PromptOptions{
				Prompt: "Min instances",
			},
			Int64PtrValidation: &cr.Int64PtrValidation{
				Required:    true,
				GreaterThan: pointer.Int64(0),
				Default:     pointer.Int64(2),
			},
		},
		{
			StructField: "MaxInstances",
			PromptOpts: &prompt.PromptOptions{
				Prompt: "Max instances",
			},
			Int64PtrValidation: &cr.Int64PtrValidation{
				Required:    true,
				GreaterThan: pointer.Int64(0),
				Default:     pointer.Int64(5),
			},
		},
	},
}

var awsCredentialsPrompt = &cr.PromptValidation{
	PromptItemValidations: []*cr.PromptItemValidation{
		{
			StructField: "AWSAccessKeyID",
			PromptOpts: &prompt.PromptOptions{
				Prompt: "Enter AWS Access Key ID",
			},
			StringValidation: &cr.StringValidation{
				Required: true,
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
			},
		},
	},
}

func validateInstanceType(instanceType string) (string, error) {
	if strings.HasSuffix(instanceType, "nano") ||
		strings.HasSuffix(instanceType, "micro") ||
		strings.HasSuffix(instanceType, "small") ||
		strings.HasSuffix(instanceType, "medium") {
		return "", errors.New("Cortex does not support nano, micro, small, or medium instances - please specify a larger instance type")
	}
	return instanceType, nil
}

func readClusterConfigYAML(clusterConfig *ClusterConfig) error {
	clusterConfigPath := files.UserPath(flagClusterConfig)
	if err := files.CheckFile(clusterConfigPath); err != nil {
		return errors.New(flagClusterConfig, "Cortex cluster configuration file does not exist")
	}

	clusterConfigBytes, err := files.ReadFileBytes(flagClusterConfig)
	if err != nil {
		return errors.Wrap(err, flagClusterConfig, userconfig.ErrorReadConfig().Error())
	}

	clusterConfigInterface, err := cr.ReadYAMLBytes(clusterConfigBytes)
	if err != nil {
		return errors.Wrap(err, flagClusterConfig, userconfig.ErrorParseConfig().Error())
	}

	errs := cr.Struct(clusterConfig, clusterConfigInterface, clusterConfigValidation)
	if errors.HasErrors(errs) {
		return errors.Wrap(errors.FirstError(errs...), flagClusterConfig)
	}

	return nil
}

func setClusterPrimaryAWSCredentials(clusterConfig *ClusterConfig) error {
	// First check env vars
	if os.Getenv("AWS_ACCESS_KEY_ID") != "" && os.Getenv("AWS_SECRET_ACCESS_KEY") != "" {
		clusterConfig.AWSAccessKeyID = os.Getenv("AWS_ACCESS_KEY_ID")
		clusterConfig.AWSSecretAccessKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
		return nil
	}
	if os.Getenv("AWS_ACCESS_KEY_ID") == "" && os.Getenv("AWS_SECRET_ACCESS_KEY") != "" {
		return errors.New("Only $AWS_SECRET_ACCESS_KEY is set; please run `export AWS_ACCESS_KEY_ID=***`")
	}
	if os.Getenv("AWS_ACCESS_KEY_ID") != "" && os.Getenv("AWS_SECRET_ACCESS_KEY") == "" {
		return errors.New("Only $AWS_ACCESS_KEY_ID is set; please run `export AWS_SECRET_ACCESS_KEY=***`")
	}

	// Next check cluster config yaml
	if clusterConfig.AWSAccessKeyID != "" && clusterConfig.AWSSecretAccessKey != "" {
		return nil
	}
	if clusterConfig.AWSAccessKeyID == "" && clusterConfig.AWSSecretAccessKey != "" {
		return errors.New(fmt.Sprintf("Only aws_secret_access_key is set in %s; please set aws_access_key_id as well", flagClusterConfig))
	}
	if clusterConfig.AWSAccessKeyID != "" && clusterConfig.AWSSecretAccessKey == "" {
		return errors.New(fmt.Sprintf("Only aws_access_key_id is set in %s; please set aws_secret_access_key as well", flagClusterConfig))
	}

	// Next check AWS CLI config file
	accessKeyID, secretAccessKey, err := aws.GetCredentialsFromCLIConfigFile()
	if err == nil {
		clusterConfig.AWSAccessKeyID = accessKeyID
		clusterConfig.AWSSecretAccessKey = secretAccessKey
		return nil
	}

	// Next check Cortex CLI config file
	cliConfig, errs := readCLIConfig()
	if !errors.HasErrors(errs) && cliConfig != nil && cliConfig.AWSAccessKeyID != "" && cliConfig.AWSSecretAccessKey != "" {
		clusterConfig.AWSAccessKeyID = cliConfig.AWSAccessKeyID
		clusterConfig.AWSSecretAccessKey = cliConfig.AWSSecretAccessKey
		return nil
	}

	// Prompt
	err = cr.ReadPrompt(clusterConfig, awsCredentialsPrompt)
	if err != nil {
		return err
	}

	return nil
}

func setClusterCortexAWSCredentials(clusterConfig *ClusterConfig) error {
	// First check env vars
	if os.Getenv("CORTEX_AWS_ACCESS_KEY_ID") != "" && os.Getenv("CORTEX_AWS_SECRET_ACCESS_KEY") != "" {
		clusterConfig.CortexAWSAccessKeyID = os.Getenv("CORTEX_AWS_ACCESS_KEY_ID")
		clusterConfig.CortexAWSSecretAccessKey = os.Getenv("CORTEX_AWS_SECRET_ACCESS_KEY")
		return nil
	}
	if os.Getenv("CORTEX_AWS_ACCESS_KEY_ID") == "" && os.Getenv("CORTEX_AWS_SECRET_ACCESS_KEY") != "" {
		return errors.New("Only $CORTEX_AWS_SECRET_ACCESS_KEY is set; please run `export CORTEX_AWS_ACCESS_KEY_ID=***`")
	}
	if os.Getenv("CORTEX_AWS_ACCESS_KEY_ID") != "" && os.Getenv("CORTEX_AWS_SECRET_ACCESS_KEY") == "" {
		return errors.New("Only $CORTEX_AWS_ACCESS_KEY_ID is set; please run `export CORTEX_AWS_SECRET_ACCESS_KEY=***`")
	}

	// Next check cluster config yaml
	if clusterConfig.CortexAWSAccessKeyID != "" && clusterConfig.CortexAWSSecretAccessKey != "" {
		return nil
	}
	if clusterConfig.CortexAWSAccessKeyID == "" && clusterConfig.CortexAWSSecretAccessKey != "" {
		return errors.New(fmt.Sprintf("Only cortex_aws_secret_access_key is set in %s; please set cortex_aws_access_key_id as well", flagClusterConfig))
	}
	if clusterConfig.CortexAWSAccessKeyID != "" && clusterConfig.CortexAWSSecretAccessKey == "" {
		return errors.New(fmt.Sprintf("Only cortex_aws_access_key_id is set in %s; please set cortex_aws_secret_access_key as well", flagClusterConfig))
	}

	// Default to primary AWS credentials
	clusterConfig.CortexAWSAccessKeyID = clusterConfig.AWSAccessKeyID
	clusterConfig.CortexAWSSecretAccessKey = clusterConfig.AWSSecretAccessKey
	return nil
}

func getInstallClusterConfig() (*ClusterConfig, error) {
	clusterConfig := &ClusterConfig{}

	if flagClusterConfig != "" {
		err := readClusterConfigYAML(clusterConfig)
		if err != nil {
			return nil, err
		}
	}

	err := cr.ReadPrompt(clusterConfig, clusterConfigPrompt)
	if err != nil {
		return nil, err
	}

	err = setClusterPrimaryAWSCredentials(clusterConfig)
	if err != nil {
		return nil, err
	}

	err = setClusterCortexAWSCredentials(clusterConfig)
	if err != nil {
		return nil, err
	}

	confirmClusterConfig(clusterConfig)

	return clusterConfig, nil
}

// This just ensures that cluster name, cluster region, and primary AWS credentails are set
func getAccessClusterConfig() (*ClusterConfig, error) {
	clusterConfig := &ClusterConfig{}

	if flagClusterConfig != "" {
		err := readClusterConfigYAML(clusterConfig)
		if err != nil {
			return nil, err
		}
	}

	err := setClusterPrimaryAWSCredentials(clusterConfig)
	if err != nil {
		return nil, err
	}

	return clusterConfig, nil
}
