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
	"context"
	"fmt"

	dockertypes "github.com/docker/docker/api/types"
	dockerclient "github.com/docker/docker/client"
	"github.com/spf13/cobra"

	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/debug"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/prompt"
	"github.com/cortexlabs/cortex/pkg/operator/api/userconfig"
)

const CORTEX_VERSION_BRANCH_STABLE = "master"

var flagCortexConfig string

func init() {
	installCmd.PersistentFlags().StringVarP(&flagCortexConfig, "config", "c", "", "path to a Cortex config file")
}

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

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "install Cortex",
	Long: `
This command installs Cortex on your AWS account.`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		clusterConfig := &ClusterConfig{}

		if flagCortexConfig != "" {
			cortexConfigPath := files.UserPath(flagCortexConfig)
			if err := files.CheckFile(cortexConfigPath); err != nil {
				errors.Exit(flagCortexConfig, "Cortex config file does not exist")
			}

			clusterConfigBytes, err := files.ReadFileBytes(flagCortexConfig)
			if err != nil {
				errors.Exit(err, flagCortexConfig, userconfig.ErrorReadConfig().Error())
			}

			clusterConfigData, err := cr.ReadYAMLBytes(clusterConfigBytes)
			if err != nil {
				errors.Exit(err, flagCortexConfig, userconfig.ErrorParseConfig().Error())
			}

			errs := cr.Struct(clusterConfig, clusterConfigData, clusterConfigValidation)

			if errors.HasErrors(errs) {
				errors.Exit(errors.FirstError(errs...), flagCortexConfig)
			}
		}

		err := cr.ReadPrompt(clusterConfig, clusterConfigPrompts)
		if err != nil {
			errors.Exit(err)
		}

		// fmt.Printf("cluster name:      %s\n", *clusterConfig.ClusterName)
		// fmt.Printf("instance type:     %s\n", *clusterConfig.NodeType)
		// fmt.Printf("min nodes:         %d\n", *clusterConfig.NodesMin)
		// fmt.Printf("max nodes:         %d\n", *clusterConfig.NodesMax)

		debug.Ppg(clusterConfig)

		str := prompt.Prompt(&prompt.PromptOptions{
			Prompt:      "Is the configuration above correct? [Y/n]",
			DefaultStr:  "Y",
			HideDefault: true,
		})
		fmt.Println(str)

		prompt.Prompt(&prompt.PromptOptions{
			Prompt: "Press [ENTER] to continue",
		})

		docker, err := dockerclient.NewClientWithOpts(dockerclient.FromEnv)
		if err != nil {
			errors.Exit(err)
		}

		docker.NegotiateAPIVersion(context.Background())

		containers, err := docker.ContainerList(context.Background(), dockertypes.ContainerListOptions{})
		if err != nil {
			panic(err)
		}

		for _, container := range containers {
			fmt.Printf("%s %s\n", container.ID[:10], container.Image)
		}

	},
}
