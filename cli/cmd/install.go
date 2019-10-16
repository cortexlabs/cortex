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
	ClusterName *string `json:"cluster_name"`
	NodeType    *string `json:"node_type"`
	NodesMin    *int64  `json:"nodes_min"`
	NodesMax    *int64  `json:"nodes_max"`
}

var clusterConfigValidation = &cr.StructValidation{
	StructFieldValidations: []*cr.StructFieldValidation{
		{
			Key:                 "cluster_name",
			StructField:         "ClusterName",
			StringPtrValidation: &cr.StringPtrValidation{},
		},
		{
			Key:                 "node_type",
			StructField:         "NodeType",
			StringPtrValidation: &cr.StringPtrValidation{},
		},
		{
			Key:         "nodes_min",
			StructField: "NodesMin",
			Int64PtrValidation: &cr.Int64PtrValidation{
				GreaterThan: pointer.Int64(0),
				Default:     pointer.Int64(1),
			},
		},
		{
			Key:         "nodes_max",
			StructField: "NodesMax",
			Int64PtrValidation: &cr.Int64PtrValidation{
				GreaterThan: pointer.Int64(0),
				Default:     pointer.Int64(3),
			},
		},
	},
}

var clusterConfigPrompts = &cr.PromptValidation{
	SkipPopulatedFields: true,
	PromptItemValidations: []*cr.PromptItemValidation{
		{
			StructField: "ClusterName",
			PromptOpts: &prompt.PromptOptions{
				Prompt: "Enter Cortex cluster name",
			},
			StringPtrValidation: &cr.StringPtrValidation{
				Required: true,
				Default:  pointer.String("cortex"),
			},
		},
		{
			StructField: "NodeType",
			PromptOpts: &prompt.PromptOptions{
				Prompt: "Enter AWS instance type",
			},
			StringPtrValidation: &cr.StringPtrValidation{
				Required: true,
				Default:  pointer.String("m5.large"),
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

		fmt.Printf("cluster name:      %s\n", *clusterConfig.ClusterName)
		fmt.Printf("instance type:     %s\n", *clusterConfig.NodeType)
		fmt.Printf("min nodes:         %d\n", *clusterConfig.NodesMin)
		fmt.Printf("max nodes:         %d\n", *clusterConfig.NodesMax)

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
