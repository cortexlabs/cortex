/*
Copyright 2020 Cortex Labs, Inc.

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

package docker

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	"github.com/cortexlabs/cortex/pkg/lib/parallel"
	dockertypes "github.com/docker/docker/api/types"
	dockerclient "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/term"
)

var _cachedDockerClient *dockerclient.Client

func createDockerClient() (*dockerclient.Client, error) {
	if _cachedDockerClient != nil {
		return _cachedDockerClient, nil
	}

	var err error
	_cachedDockerClient, err = dockerclient.NewClientWithOpts(dockerclient.FromEnv)
	if err != nil {
		return nil, WrapDockerError(err)
	}

	_cachedDockerClient.NegotiateAPIVersion(context.Background())
	return _cachedDockerClient, nil
}

func GetDockerClient() (*dockerclient.Client, error) {
	dockerClient, err := createDockerClient()
	if err != nil {
		return nil, err
	}

	if _, err := dockerClient.Info(context.Background()); err != nil {
		return nil, WrapDockerError(err)
	}

	return dockerClient, nil
}

func MustDockerClient() *dockerclient.Client {
	dockerClient, err := createDockerClient()
	if err != nil {
		exit.Error(err)
	}

	if _, err := dockerClient.Info(context.Background()); err != nil {
		exit.Error(WrapDockerError(err))
	}

	return dockerClient
}

func WrapDockerError(err error) error {
	if dockerclient.IsErrConnectionFailed(err) {
		return ErrorConnectToDockerDaemon()
	}

	if strings.Contains(strings.ToLower(err.Error()), "permission denied") {
		return ErrorDockerPermissions(err)
	}

	return errors.WithStack(err)
}

func PullImage(managerImage string) error {
	docker, err := GetDockerClient()
	if err != nil {
		return err
	}

	images, err := docker.ImageList(context.Background(), dockertypes.ImageListOptions{})
	if err != nil {
		return WrapDockerError(err)
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
		return WrapDockerError(err)
	}
	defer pullOutput.Close()

	termFd, isTerm := term.GetFdInfo(os.Stderr)
	jsonmessage.DisplayJSONMessagesStream(pullOutput, os.Stderr, termFd, isTerm, nil)
	fmt.Println()

	return nil
}

func StreamDockerLogs(containerID string, containerIDs ...string) error {
	containerIDs = append([]string{containerID}, containerIDs...)

	docker, err := GetDockerClient()
	if err != nil {
		return err
	}

	// c := make(chan os.Signal, 1)
	// signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	// caughtCtrlC := false
	// go func() {
	// 	<-c
	// 	caughtCtrlC = true
	// 	exit.Error(ErrorDockerCtrlC())
	// }()

	fns := make([]func() error, len(containerIDs))
	for i, containerID := range containerIDs {
		fns[i] = StreamDockerLogsFn(containerID, docker)
	}

	err = parallel.RunFirstErr(fns[0], fns[1:]...)

	if err != nil {
		return WrapDockerError(err)
	}

	// // Let the ctrl+c handler run its course
	// if caughtCtrlC {
	// 	time.Sleep(5 * time.Second)
	// }

	return nil
}

func StreamDockerLogsFn(containerID string, docker *dockerclient.Client) func() error {
	return func() error {
		// Use ContainerLogs() so lines are only printed once they end in \n
		logsOutput, err := docker.ContainerLogs(context.Background(), containerID, dockertypes.ContainerLogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Follow:     true,
		})
		if err != nil {
			return WrapDockerError(err)
		}

		_, err = io.Copy(os.Stdout, logsOutput)
		if err != nil && err != io.EOF {
			return errors.WithStack(err)
		}

		return nil
	}
}
