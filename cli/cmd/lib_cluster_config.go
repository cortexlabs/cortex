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
	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/prompt"
	"github.com/cortexlabs/cortex/pkg/operator/api/userconfig"
)

type ClusterConfig struct {
	AWSAccessKeyID           *string `json:"aws_access_key_id"`
	AWSSecretAccessKey       *string `json:"aws_secret_access_key"`
	CortexAWSAccessKeyID     *string `json:"cortex_aws_access_key_id"`
	CortexAWSSecretAccessKey *string `json:"cortex_aws_secret_access_key"`
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
	ImageTFApi               string  `json:"image_tf_api"`
	ImageTFServeGpu          string  `json:"image_tf_serve_gpu"`
	ImageOnnxServe           string  `json:"image_onnx_serve"`
	ImageOnnxServeGpu        string  `json:"image_onnx_serve_gpu"`
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
			StructField:         "InstanceType",
			StringPtrValidation: &cr.StringPtrValidation{},
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
				Default: "cortex",
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
				Default: "cortex",
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
				Default: "cortexlabs/manager",
			},
		},
		{
			StructField: "ImageFluentd",
			StringValidation: &cr.StringValidation{
				Default: "cortexlabs/fluentd",
			},
		},
		{
			StructField: "ImageStatsd",
			StringValidation: &cr.StringValidation{
				Default: "cortexlabs/statsd",
			},
		},
		{
			StructField: "ImageOperator",
			StringValidation: &cr.StringValidation{
				Default: "cortexlabs/operator",
			},
		},
		{
			StructField: "ImageTFServe",
			StringValidation: &cr.StringValidation{
				Default: "cortexlabs/tf-serve",
			},
		},
		{
			StructField: "ImageTFApi",
			StringValidation: &cr.StringValidation{
				Default: "cortexlabs/tf-api",
			},
		},
		{
			StructField: "ImageTFServeGpu",
			StringValidation: &cr.StringValidation{
				Default: "cortexlabs/tf-serve-gpu",
			},
		},
		{
			StructField: "ImageOnnxServe",
			StringValidation: &cr.StringValidation{
				Default: "cortexlabs/onnx-serve",
			},
		},
		{
			StructField: "ImageOnnxServeGpu",
			StringValidation: &cr.StringValidation{
				Default: "cortexlabs/onnx-serve-gpu",
			},
		},
		{
			StructField: "ImageClusterAutoscaler",
			StringValidation: &cr.StringValidation{
				Default: "cortexlabs/cluster-autoscaler",
			},
		},
		{
			StructField: "ImageNvidia",
			StringValidation: &cr.StringValidation{
				Default: "cortexlabs/nvidia",
			},
		},
		{
			StructField: "ImageMetricsServer",
			StringValidation: &cr.StringValidation{
				Default: "cortexlabs/metrics-server",
			},
		},
		{
			StructField: "ImageIstioCitadel",
			StringValidation: &cr.StringValidation{
				Default: "cortexlabs/istio-citadel",
			},
		},
		{
			StructField: "ImageIstioGalley",
			StringValidation: &cr.StringValidation{
				Default: "cortexlabs/istio-galley",
			},
		},
		{
			StructField: "ImageIstioPilot",
			StringValidation: &cr.StringValidation{
				Default: "cortexlabs/istio-pilot",
			},
		},
		{
			StructField: "ImageIstioProxy",
			StringValidation: &cr.StringValidation{
				Default: "cortexlabs/istio-proxy",
			},
		},
		{
			StructField: "ImageDownloader",
			StringValidation: &cr.StringValidation{
				Default: "cortexlabs/downloader",
			},
		},
	},
}

var clusterConfigPrompts = &cr.PromptValidation{
	SkipPopulatedFields: true,
	PromptItemValidations: []*cr.PromptItemValidation{
		{
			StructField: "InstanceType",
			PromptOpts: &prompt.PromptOptions{
				Prompt: "AWS instance type",
			},
			StringPtrValidation: &cr.StringPtrValidation{
				Required: true,
				Default:  pointer.String("m5.large"),
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

func readClusterConfigYAML() (interface{}, error) {
	clusterConfigPath := files.UserPath(flagClusterConfig)
	if err := files.CheckFile(clusterConfigPath); err != nil {
		return nil, errors.New(flagClusterConfig, "Cortex cluster configuration file does not exist")
	}

	clusterConfigBytes, err := files.ReadFileBytes(flagClusterConfig)
	if err != nil {
		return nil, errors.Wrap(err, flagClusterConfig, userconfig.ErrorReadConfig().Error())
	}

	clusterConfigInterface, err := cr.ReadYAMLBytes(clusterConfigBytes)
	if err != nil {
		return nil, errors.Wrap(err, flagClusterConfig, userconfig.ErrorParseConfig().Error())
	}

	return clusterConfigInterface, nil
}

func getClusterConfig(includeAWSCredentials bool) (*ClusterConfig, error) {
	clusterConfig := &ClusterConfig{}

	if flagClusterConfig != "" {
		clusterConfigInterface, err := readClusterConfigYAML()
		if err != nil {
			return nil, err
		}

		errs := cr.Struct(clusterConfig, clusterConfigInterface, clusterConfigValidation)
		if errors.HasErrors(errs) {
			return nil, errors.Wrap(errors.FirstError(errs...), flagClusterConfig)
		}
	}

	err := cr.ReadPrompt(clusterConfig, clusterConfigPrompts)
	if err != nil {
		return nil, err
	}

	return clusterConfig, nil
}
