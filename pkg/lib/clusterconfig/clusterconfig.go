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

package clusterconfig

import (
	"strings"

	"github.com/cortexlabs/cortex/pkg/consts"
	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/prompt"
)

type ClusterConfig struct {
	InstanceType           *string `json:"instance_type" yaml:"instance_type"`
	MinInstances           *int64  `json:"min_instances" yaml:"min_instances"`
	MaxInstances           *int64  `json:"max_instances" yaml:"max_instances"`
	ClusterName            string  `json:"cluster_name" yaml:"cluster_name"`
	Region                 string  `json:"region" yaml:"region"`
	Bucket                 string  `json:"bucket" yaml:"bucket"`
	LogGroup               string  `json:"log_group" yaml:"log_group"`
	Telemetry              bool    `json:"telemetry" yaml:"telemetry"`
	ImageManager           string  `json:"image_manager" yaml:"image_manager"`
	ImageFluentd           string  `json:"image_fluentd" yaml:"image_fluentd"`
	ImageStatsd            string  `json:"image_statsd" yaml:"image_statsd"`
	ImageOperator          string  `json:"image_operator" yaml:"image_operator"`
	ImageTFServe           string  `json:"image_tf_serve" yaml:"image_tf_serve"`
	ImageTFAPI             string  `json:"image_tf_api" yaml:"image_tf_api"`
	ImageTFServeGPU        string  `json:"image_tf_serve_gpu" yaml:"image_tf_serve_gpu"`
	ImageOnnxServe         string  `json:"image_onnx_serve" yaml:"image_onnx_serve"`
	ImageOnnxServeGPU      string  `json:"image_onnx_serve_gpu" yaml:"image_onnx_serve_gpu"`
	ImageClusterAutoscaler string  `json:"image_cluster_autoscaler" yaml:"image_cluster_autoscaler"`
	ImageNvidia            string  `json:"image_nvidia" yaml:"image_nvidia"`
	ImageMetricsServer     string  `json:"image_metrics_server" yaml:"image_metrics_server"`
	ImageIstioCitadel      string  `json:"image_istio_citadel" yaml:"image_istio_citadel"`
	ImageIstioGalley       string  `json:"image_istio_galley" yaml:"image_istio_galley"`
	ImageIstioPilot        string  `json:"image_istio_pilot" yaml:"image_istio_pilot"`
	ImageIstioProxy        string  `json:"image_istio_proxy" yaml:"image_istio_proxy"`
	ImageDownloader        string  `json:"image_downloader" yaml:"image_downloader"`
}

type InternalClusterConfig struct {
	ClusterConfig
	ID                string `json:"id"`
	APIVersion        string `json:"api_version"`
	OperatorInCluster bool   `json:"operator_in_cluster"`
}

var Validation = &cr.StructValidation{
	StructFieldValidations: []*cr.StructFieldValidation{
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
		// Extra keys from user cluster config files
		{
			Key: "aws_access_key_id",
			Nil: true,
		},
		{
			Key: "aws_secret_access_key",
			Nil: true,
		},
		{
			Key: "cortex_aws_access_key_id",
			Nil: true,
		},
		{
			Key: "cortex_aws_secret_access_key",
			Nil: true,
		},
	},
}

var PromptValidation = &cr.PromptValidation{
	SkipPopulatedFields: true,
	PromptItemValidations: []*cr.PromptItemValidation{
		{
			StructField: "InstanceType",
			PromptOpts: &prompt.Options{
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
			PromptOpts: &prompt.Options{
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
			PromptOpts: &prompt.Options{
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

func validateInstanceType(instanceType string) (string, error) {
	if strings.HasSuffix(instanceType, "nano") ||
		strings.HasSuffix(instanceType, "micro") ||
		strings.HasSuffix(instanceType, "small") ||
		strings.HasSuffix(instanceType, "medium") {
		return "", errors.New("Cortex does not support nano, micro, small, or medium instances - please specify a larger instance type")
	}
	return instanceType, nil
}
