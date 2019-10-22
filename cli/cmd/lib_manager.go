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
	"context"
	"io"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"github.com/cortexlabs/cortex/pkg/lib/clusterconfig"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/yaml"
	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	dockerclient "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/term"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
)

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

func pullManager(clusterConfig *clusterconfig.ClusterConfig) error {
	docker, err := getDockerClient()
	if err != nil {
		return err
	}

	images, err := docker.ImageList(context.Background(), dockertypes.ImageListOptions{})
	if err != nil {
		return err
	}

	for _, image := range images {
		for _, tag := range image.RepoTags {
			if tag == clusterConfig.ImageManager {
				return nil
			}
		}
	}

	pullOutput, err := docker.ImagePull(context.Background(), clusterConfig.ImageManager, dockertypes.ImagePullOptions{})
	if err != nil {
		return err
	}
	defer pullOutput.Close()

	termFd, isTerm := term.GetFdInfo(os.Stderr)
	jsonmessage.DisplayJSONMessagesStream(pullOutput, os.Stderr, termFd, isTerm, nil)

	return nil
}

func runManagerCommand(entrypoint string, clusterConfig *clusterconfig.ClusterConfig, awsCreds *AWSCredentials) error {
	docker, err := getDockerClient()
	if err != nil {
		return err
	}

	pullManager(clusterConfig)

	clusterConfigBytes, err := yaml.Marshal(clusterConfig)
	if err != nil {
		return err
	}
	if err := files.WriteFile(clusterConfigBytes, path.Join(localDir, "cluster.yaml")); err != nil {
		return err
	}

	containerConfig := &container.Config{
		Image:        clusterConfig.ImageManager,
		Entrypoint:   []string{"/bin/bash", "-c"},
		Cmd:          []string{"eval $(python /root/cluster_config_env.py /.cortex/cluster.yaml) && " + entrypoint},
		Tty:          true,
		AttachStdout: true,
		AttachStderr: true,
		Env: []string{
			"AWS_ACCESS_KEY_ID=" + awsCreds.AWSAccessKeyID,
			"AWS_SECRET_ACCESS_KEY=" + awsCreds.AWSSecretAccessKey,
			"CORTEX_AWS_ACCESS_KEY_ID=" + awsCreds.CortexAWSAccessKeyID,
			"CORTEX_AWS_SECRET_ACCESS_KEY=" + awsCreds.CortexAWSSecretAccessKey,
		},
	}

	hostConfig := &container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: localDir,
				Target: "/.cortex",
			},
		},
	}

	containerInfo, err := docker.ContainerCreate(context.Background(), containerConfig, hostConfig, nil, "")
	if err != nil {
		errors.Exit(err)
	}

	removeContainer := func() {
		docker.ContainerRemove(context.Background(), containerInfo.ID, dockertypes.ContainerRemoveOptions{
			RemoveVolumes: true,
			Force:         true,
		})
	}

	defer removeContainer()

	// Make sure to remove container immediately on ctrl+c
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	caughtCtrlC := false
	go func() {
		<-c
		caughtCtrlC = true
		removeContainer()
		os.Exit(1)
	}()

	err = docker.ContainerStart(context.Background(), containerInfo.ID, dockertypes.ContainerStartOptions{})
	if err != nil {
		return err
	}

	logOpts := dockertypes.ContainerLogsOptions{
		ShowStderr: true,
		ShowStdout: true,
		Follow:     true,
	}
	logsOutput, err := docker.ContainerLogs(context.Background(), containerInfo.ID, logOpts)
	if err != nil {
		return err
	}
	defer logsOutput.Close()

	_, err = io.Copy(os.Stdout, logsOutput)
	if err != nil {
		return err
	}

	// Let the ctrl+C hanlder run its course
	if caughtCtrlC {
		time.Sleep(time.Second)
		return nil
	}

	return nil
}
