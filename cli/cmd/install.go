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

	"github.com/cortexlabs/cortex/pkg/lib/debug"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/prompt"
)

func init() {
	addClusterConfigFlag(installCmd)
}

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "install Cortex",
	Long: `
This command installs Cortex on your AWS account.`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		clusterConfig, err := getClusterConfig(true)
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
