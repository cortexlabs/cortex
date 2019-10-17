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
	"os"
	"strings"

	dockertypes "github.com/docker/docker/api/types"
	dockerclient "github.com/docker/docker/client"
	"github.com/spf13/cobra"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/prompt"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
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

		confirmClusterConfig(clusterConfig)

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

func confirmClusterConfig(clusterConfig *ClusterConfig) {
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
	fmt.Printf("AWS access key ID: %s\n", s.MaskString(clusterConfig.AWSAccessKeyID, 4))
	if clusterConfig.CortexAWSAccessKeyID != clusterConfig.AWSAccessKeyID {
		fmt.Printf("AWS access key ID: %s\n (cortex)", s.MaskString(clusterConfig.CortexAWSAccessKeyID, 4))
	}
	fmt.Println()

	for true {
		str := prompt.Prompt(&prompt.PromptOptions{
			Prompt:      "Is the configuration above correct? [y/n]",
			HideDefault: true,
		})

		if strings.ToLower(str) == "y" {
			return
		}

		if strings.ToLower(str) == "n" {
			os.Exit(1)
		}

		fmt.Println("please enter \"y\" or \"n\"")
	}
}
