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

// cx cluster up/down

package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	dockerclient "github.com/docker/docker/client"
	"github.com/spf13/cobra"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/prompt"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
)

var flagClusterConfig string

func init() {
	addClusterConfigFlag(upCmd)
	addClusterConfigFlag(downCmd)
	clusterCmd.AddCommand(upCmd)
	clusterCmd.AddCommand(downCmd)
}

func addClusterConfigFlag(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVarP(&flagClusterConfig, "config", "c", "", "path to a Cortex cluster configuration file")
}

var clusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "manage a Cortex cluster",
	Long:  "Manage a Cortex cluster",
}

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "spin up a Cortex cluster",
	Long: `
This command spins up a Cortex cluster on your AWS account.`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		clusterConfig, err := getClusterConfig(true)
		if err != nil {
			errors.Exit(err)
		}

		confirmClusterConfig(clusterConfig)

		err = installEKS(clusterConfig)
		if err != nil {
			errors.Exit(err)
		}

		err = installCortex(clusterConfig)
		if err != nil {
			errors.Exit(err)
		}
	},
}

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "get information about a Cortex cluster",
	Long: `
This command gets information about a Cortex cluster.`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		clusterConfig, err := getClusterConfig(true)
		if err != nil {
			errors.Exit(err)
		}

		confirmClusterConfig(clusterConfig) // TODO just need subset, read from file saved on install?

		err = clusterInfo(clusterConfig)
		if err != nil {
			errors.Exit(err)
		}
	},
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "update a Cortex cluster",
	Long: `
This command updates a Cortex cluster.`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		clusterConfig, err := getClusterConfig(true)
		if err != nil {
			errors.Exit(err)
		}

		confirmClusterConfig(clusterConfig)

		err = installCortex(clusterConfig)
		if err != nil {
			errors.Exit(err)
		}
	},
}

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "spin down a Cortex cluster",
	Long: `
This command spins down a Cortex cluster.`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		clusterConfig, err := getClusterConfig(true)
		if err != nil {
			errors.Exit(err)
		}

		confirmClusterConfig(clusterConfig) // TODO just need subset, read from file saved on install?
	},
}

var cachedDocker *dockerclient.Client

func getDockerClient() (*dockerclient.Client, error) {
	if cachedDocker != nil {
		return cachedDocker, nil
	}

	var err error
	cachedDocker, err = dockerclient.NewClientWithOpts(dockerclient.FromEnv)
	if err != nil {
		return nil, err
	}

	cachedDocker.NegotiateAPIVersion(context.Background())
	return cachedDocker, nil
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

	return

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
		fmt.Println()
	}
}

func installEKS(clusterConfig *ClusterConfig) error {
	docker, err := getDockerClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	// pullOut, err := docker.ImagePull(ctx, "debian:bullseye-20191014", dockertypes.ImagePullOptions{})
	// if err != nil {
	// 	return err
	// }
	// defer pullOut.Close()

	// io.Copy(os.Stdout, pullOut)

	containerConfig := &container.Config{
		Image:        clusterConfig.ImageManager,
		Entrypoint:   []string{"/bin/bash", "-c"},
		Cmd:          []string{"ls && sleep 5 && ls"},
		Tty:          true,
		AttachStdout: true,
		AttachStderr: true,
	}
	containerInfo, err := docker.ContainerCreate(ctx, containerConfig, nil, nil, "")
	if err != nil {
		errors.Exit(err)
	}

	cleanup := func() {
		fmt.Println("CLEANUP")
		docker.ContainerRemove(ctx, containerInfo.ID, dockertypes.ContainerRemoveOptions{
			RemoveVolumes: true,
			Force:         true,
		})
		fmt.Println("done cleanup")
	}

	defer cleanup()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("caught")
		cleanup()
		fmt.Println("trying to exit")
		os.Exit(1)
	}()

	err = docker.ContainerStart(ctx, containerInfo.ID, dockertypes.ContainerStartOptions{})
	if err != nil {
		return err
	}

	logOpts := dockertypes.ContainerLogsOptions{
		ShowStderr: true,
		ShowStdout: true,
		Follow:     true,
	}
	logsOut, err := docker.ContainerLogs(ctx, containerInfo.ID, logOpts)
	if err != nil {
		return err
	}
	defer logsOut.Close()

	scanner := bufio.NewScanner(logsOut)
	for scanner.Scan() {
		fmt.Println(scanner.Text())
	}

	// io.Copy(os.Stdout, logsOut)

	fmt.Println("HERE")

	return nil
}

func installCortex(clusterConfig *ClusterConfig) error {
	docker, err := getDockerClient()
	if err != nil {
		return err
	}

	containers, err := docker.ContainerList(context.Background(), dockertypes.ContainerListOptions{})
	if err != nil {
		panic(err)
	}

	for _, container := range containers {
		fmt.Printf("%s %s\n", container.ID[:10], container.Image)
	}

	return nil
}

func clusterInfo(clusterConfig *ClusterConfig) error {
	docker, err := getDockerClient()
	if err != nil {
		return err
	}

	containers, err := docker.ContainerList(context.Background(), dockertypes.ContainerListOptions{})
	if err != nil {
		panic(err)
	}

	for _, container := range containers {
		fmt.Printf("%s %s\n", container.ID[:10], container.Image)
	}

	return nil
}
