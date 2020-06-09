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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/cron"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	"github.com/cortexlabs/cortex/pkg/lib/parallel"
	"github.com/cortexlabs/cortex/pkg/lib/print"
	"github.com/cortexlabs/cortex/pkg/lib/slices"
	dockertypes "github.com/docker/docker/api/types"
	dockerclient "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/term"
)

var NoAuth string

var _cachedClient *Client

type Client struct {
	*dockerclient.Client
	Info dockertypes.Info
}

func init() {
	NoAuth, _ = EncodeAuthConfig(dockertypes.AuthConfig{})
}

func GetDockerClient() (*Client, error) {
	if _cachedClient != nil {
		return _cachedClient, nil
	}

	baseClient, err := dockerclient.NewClientWithOpts(dockerclient.FromEnv)
	if err != nil {
		return nil, WrapDockerError(err)
	}

	baseClient.NegotiateAPIVersion(context.Background())

	info, err := baseClient.Info(context.Background())
	if err != nil {
		return nil, WrapDockerError(err)
	}

	_cachedClient = &Client{
		Client: baseClient,
		Info:   info,
	}

	return _cachedClient, nil
}

func MustDockerClient() *Client {
	dockerClient, err := GetDockerClient()
	if err != nil {
		exit.Error(err)
	}

	return dockerClient
}

func AWSAuthConfig(awsClient *aws.Client) (string, error) {
	dockerClient, err := GetDockerClient()
	if err != nil {
		return "", err
	}

	ecrAuthConfig, err := awsClient.GetECRAuthConfig()
	if err != nil {
		return "", err
	}

	auth := dockertypes.AuthConfig{
		Username:      ecrAuthConfig.Username,
		Password:      ecrAuthConfig.AccessToken,
		ServerAddress: ecrAuthConfig.ProxyEndpoint,
	}

	_, err = dockerClient.RegistryLogin(context.Background(), auth)
	if err != nil {
		return "", err
	}

	authConfig, err := EncodeAuthConfig(auth)
	if err != nil {
		return "", err
	}

	return authConfig, nil
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

type PullVerbosity int

const (
	NoPrint PullVerbosity = iota
	PrintDots
	PrintProgressBars
)

func PullImage(image string, encodedAuthConfig string, pullVerbosity PullVerbosity) (bool, error) {
	dockerClient, err := GetDockerClient()
	if err != nil {
		return false, err
	}

	if err := CheckLocalImageAccessible(dockerClient, image); err == nil {
		return false, nil
	}

	pullOutput, err := dockerClient.ImagePull(context.Background(), image, dockertypes.ImagePullOptions{
		RegistryAuth: encodedAuthConfig,
	})
	if err != nil {
		return false, WrapDockerError(err)
	}
	defer pullOutput.Close()

	switch pullVerbosity {
	case PrintProgressBars:
		termFd, isTerm := term.GetFdInfo(os.Stderr)
		jsonmessage.DisplayJSONMessagesStream(pullOutput, os.Stderr, termFd, isTerm, nil)
		fmt.Println()
	case PrintDots:
		fmt.Printf("￮ downloading docker image %s ", image)
		defer fmt.Print(" ✓\n")
		dotCron := cron.Run(print.Dot, nil, 2*time.Second)
		defer dotCron.Cancel()
		// wait until the pull has completed
		if _, err := ioutil.ReadAll(pullOutput); err != nil {
			return false, err
		}
	default:
		// wait until the pull has completed
		if _, err := ioutil.ReadAll(pullOutput); err != nil {
			return false, err
		}
	}

	return true, nil
}

func StreamDockerLogs(containerID string, containerIDs ...string) error {
	containerIDs = append([]string{containerID}, containerIDs...)

	dockerClient, err := GetDockerClient()
	if err != nil {
		return err
	}

	fns := make([]func() error, len(containerIDs))
	for i, containerID := range containerIDs {
		fns[i] = StreamDockerLogsFn(containerID, dockerClient)
	}

	err = parallel.RunFirstErr(fns[0], fns[1:]...)

	if err != nil {
		return WrapDockerError(err)
	}

	return nil
}

func StreamDockerLogsFn(containerID string, dockerClient *Client) func() error {
	return func() error {
		// Use ContainerLogs() so lines are only printed once they end in \n
		logsOutput, err := dockerClient.ContainerLogs(context.Background(), containerID, dockertypes.ContainerLogsOptions{
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

func EncodeAuthConfig(authConfig dockertypes.AuthConfig) (string, error) {
	encoded, err := json.Marshal(authConfig)
	if err != nil {
		return "", errors.Wrap(err, "failed to encode docker login credentials")
	}
	registryAuth := base64.URLEncoding.EncodeToString(encoded)
	return registryAuth, nil
}

func CheckImageAccessible(dockerClient *Client, dockerImage, authConfig string) error {
	if _, err := dockerClient.DistributionInspect(context.Background(), dockerImage, authConfig); err != nil {
		return ErrorImageInaccessible(dockerImage, err)
	}
	return nil
}

func CheckLocalImageAccessible(dockerClient *Client, dockerImage string) error {
	images, err := dockerClient.ImageList(context.Background(), dockertypes.ImageListOptions{})
	if err != nil {
		return WrapDockerError(err)
	}

	// in docker, missing tag implies "latest"
	if ExtractImageTag(dockerImage) == "" {
		dockerImage = fmt.Sprintf("%s:latest", dockerImage)
	}

	for _, image := range images {
		if slices.HasString(image.RepoTags, dockerImage) {
			return nil
		}
	}
	return ErrorImageInaccessible(dockerImage, nil)
}

func ExtractImageTag(dockerImage string) string {
	if colonIndex := strings.LastIndex(dockerImage, ":"); colonIndex != -1 {
		return dockerImage[colonIndex+1:]
	}
	return ""
}
