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
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cortexlabs/yaml"
	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	dockerclient "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/term"

	"github.com/cortexlabs/cortex/pkg/lib/clusterconfig"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
)

var cachedDockerClient *dockerclient.Client

func getDockerClient() (*dockerclient.Client, error) {
	if cachedDockerClient != nil {
		return cachedDockerClient, nil
	}

	var err error
	cachedDockerClient, err = dockerclient.NewClientWithOpts(dockerclient.FromEnv)
	if err != nil {
		return nil, wrapDockerError(err)
	}

	cachedDockerClient.NegotiateAPIVersion(context.Background())
	return cachedDockerClient, nil
}

func wrapDockerError(err error) error {
	if dockerclient.IsErrConnectionFailed(err) {
		return errors.New("Unable to connect to the Docker daemon, please confirm Docker is running")
	}

	return errors.WithStack(err)
}

func checkDockerRunning() error {
	docker, err := getDockerClient()
	if err != nil {
		return err
	}

	if _, err := docker.Info(context.Background()); err != nil {
		return wrapDockerError(err)
	}

	return nil
}

func pullManager(clusterConfig *clusterconfig.ClusterConfig) error {
	docker, err := getDockerClient()
	if err != nil {
		return err
	}

	images, err := docker.ImageList(context.Background(), dockertypes.ImageListOptions{})
	if err != nil {
		return wrapDockerError(err)
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
		return wrapDockerError(err)
	}
	defer pullOutput.Close()

	termFd, isTerm := term.GetFdInfo(os.Stderr)
	jsonmessage.DisplayJSONMessagesStream(pullOutput, os.Stderr, termFd, isTerm, nil)
	fmt.Println()

	return nil
}

func runManagerCommand(entrypoint string, clusterConfig *clusterconfig.ClusterConfig, awsCreds *AWSCredentials) (string, error) {
	docker, err := getDockerClient()
	if err != nil {
		return "", err
	}

	err = pullManager(clusterConfig)
	if err != nil {
		return "", err
	}

	clusterConfigBytes, err := yaml.Marshal(clusterConfig)
	if err != nil {
		return "", errors.WithStack(err)
	}
	if err := files.WriteFile(clusterConfigBytes, cachedClusterConfigPath); err != nil {
		return "", err
	}

	containerConfig := &container.Config{
		Image:        clusterConfig.ImageManager,
		Entrypoint:   []string{"/bin/bash", "-c"},
		Cmd:          []string{"sleep 0.1 && eval $(python /root/cluster_config_env.py /.cortex/cluster.yaml) && " + entrypoint},
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
		return "", wrapDockerError(err)
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
		return "", wrapDockerError(err)
	}

	// There is a slight delay between the container starting at attaching to it, hence the sleep in Cmd
	logsOutput, err := docker.ContainerAttach(context.Background(), containerInfo.ID, dockertypes.ContainerAttachOptions{
		Stream: true,
		Stdout: true,
		Stderr: true,
	})
	if err != nil {
		return "", wrapDockerError(err)
	}
	defer logsOutput.Close()

	var outputBuffer bytes.Buffer
	tee := io.TeeReader(logsOutput.Reader, &outputBuffer)

	_, err = io.Copy(os.Stdout, tee)
	if err != nil {
		return "", errors.WithStack(err)
	}

	output := outputBuffer.String()

	// Let the ctrl+C hanlder run its course
	if caughtCtrlC {
		time.Sleep(time.Second)
		return output, nil
	}

	return output, nil
}
