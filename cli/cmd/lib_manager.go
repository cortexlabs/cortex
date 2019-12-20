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

	"github.com/cortexlabs/cortex/pkg/lib/clusterconfig"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/yaml"
	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	dockerclient "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/term"
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

func pullManager(managerImage string) error {
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
			if tag == managerImage {
				return nil
			}
		}
	}

	pullOutput, err := docker.ImagePull(context.Background(), managerImage, dockertypes.ImagePullOptions{})
	if err != nil {
		return wrapDockerError(err)
	}
	defer pullOutput.Close()

	termFd, isTerm := term.GetFdInfo(os.Stderr)
	jsonmessage.DisplayJSONMessagesStream(pullOutput, os.Stderr, termFd, isTerm, nil)
	fmt.Println()

	return nil
}

func runManager(containerConfig *container.Config) (string, *int, error) {
	docker, err := getDockerClient()
	if err != nil {
		return "", nil, err
	}

	err = pullManager(containerConfig.Image)
	if err != nil {
		return "", nil, err
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
		return "", nil, wrapDockerError(err)
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
		exit.ErrorNoPrintNoTelemetry()
	}()

	err = docker.ContainerStart(context.Background(), containerInfo.ID, dockertypes.ContainerStartOptions{})
	if err != nil {
		return "", nil, wrapDockerError(err)
	}

	// There is a slight delay between the container starting at attaching to it, hence the sleep in Cmd
	logsOutput, err := docker.ContainerAttach(context.Background(), containerInfo.ID, dockertypes.ContainerAttachOptions{
		Stream: true,
		Stdout: true,
		Stderr: true,
	})
	if err != nil {
		return "", nil, wrapDockerError(err)
	}
	defer logsOutput.Close()

	var outputBuffer bytes.Buffer
	tee := io.TeeReader(logsOutput.Reader, &outputBuffer)

	_, err = io.Copy(os.Stdout, tee)
	if err != nil {
		return "", nil, errors.WithStack(err)
	}

	output := outputBuffer.String()

	// Let the ctrl+C handler run its course
	if caughtCtrlC {
		time.Sleep(5 * time.Second)
	}

	info, err := docker.ContainerInspect(context.Background(), containerInfo.ID)
	if err != nil {
		return "", nil, errors.WithStack(err)
	}

	if info.State.Running {
		return output, nil, nil
	}

	return output, &info.State.ExitCode, nil
}

func runManagerUpdateCommand(entrypoint string, clusterConfig *clusterconfig.Config, awsCreds *AWSCredentials) (string, *int, error) {
	clusterConfigBytes, err := yaml.Marshal(clusterConfig)
	if err != nil {
		return "", nil, errors.WithStack(err)
	}

	cachedConfigPath := cachedClusterConfigPath(clusterConfig.ClusterName, *clusterConfig.Region)
	if err := files.WriteFile(clusterConfigBytes, cachedConfigPath); err != nil {
		return "", nil, err
	}

	mountedConfigPath := mountedClusterConfigPath(clusterConfig.ClusterName, *clusterConfig.Region)

	containerConfig := &container.Config{
		Image:        clusterConfig.ImageManager,
		Entrypoint:   []string{"/bin/bash", "-c"},
		Cmd:          []string{fmt.Sprintf("sleep 0.1 && eval $(python /root/cluster_config_env.py %s) && %s", mountedConfigPath, entrypoint)},
		Tty:          true,
		AttachStdout: true,
		AttachStderr: true,
		Env: []string{
			"CORTEX_ENVIRONMENT=" + flagEnv,
			"AWS_ACCESS_KEY_ID=" + awsCreds.AWSAccessKeyID,
			"AWS_SECRET_ACCESS_KEY=" + awsCreds.AWSSecretAccessKey,
			"CORTEX_AWS_ACCESS_KEY_ID=" + awsCreds.CortexAWSAccessKeyID,
			"CORTEX_AWS_SECRET_ACCESS_KEY=" + awsCreds.CortexAWSSecretAccessKey,
			"CORTEX_TELEMETRY_DISABLE=" + os.Getenv("CORTEX_TELEMETRY_DISABLE"),
			"CORTEX_TELEMETRY_SENTRY_DSN=" + os.Getenv("CORTEX_TELEMETRY_SENTRY_DSN"),
			"CORTEX_TELEMETRY_SEGMENT_WRITE_KEY=" + os.Getenv("CORTEX_TELEMETRY_SEGMENT_WRITE_KEY"),
			"CORTEX_CLUSTER_CONFIG_FILE=" + mountedConfigPath,
		},
	}

	output, exitCode, err := runManager(containerConfig)
	if err != nil {
		return "", nil, err
	}

	return output, exitCode, nil
}

func runManagerAccessCommand(entrypoint string, accessConfig clusterconfig.AccessConfig, awsCreds *AWSCredentials) (string, *int, error) {
	containerConfig := &container.Config{
		Image:        accessConfig.ImageManager,
		Entrypoint:   []string{"/bin/bash", "-c"},
		Cmd:          []string{"sleep 0.1 && " + entrypoint},
		Tty:          true,
		AttachStdout: true,
		AttachStderr: true,
		Env: []string{
			"CORTEX_ENVIRONMENT=" + flagEnv,
			"AWS_ACCESS_KEY_ID=" + awsCreds.AWSAccessKeyID,
			"AWS_SECRET_ACCESS_KEY=" + awsCreds.AWSSecretAccessKey,
			"CORTEX_AWS_ACCESS_KEY_ID=" + awsCreds.CortexAWSAccessKeyID,
			"CORTEX_AWS_SECRET_ACCESS_KEY=" + awsCreds.CortexAWSSecretAccessKey,
			"CORTEX_CLUSTER_NAME=" + *accessConfig.ClusterName,
			"CORTEX_REGION=" + *accessConfig.Region,
		},
	}

	output, exitCode, err := runManager(containerConfig)
	if err != nil {
		return "", nil, err
	}

	return output, exitCode, nil
}
