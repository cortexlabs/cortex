/*
Copyright 2021 Cortex Labs, Inc.

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
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/cortexlabs/cortex/cli/lib/routines"
	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/archive"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/docker"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
	"github.com/cortexlabs/yaml"
	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
)

type dockerCopyFromPath struct {
	containerPath string
	localDir      string
}

type dockerCopyToPath struct {
	input         *archive.Input
	containerPath string
}

func runManager(containerConfig *container.Config, addNewLineAfterPull bool, copyToPaths []dockerCopyToPath, copyFromPaths []dockerCopyFromPath) (string, *int, error) {
	containerConfig.Env = append(containerConfig.Env, "CORTEX_CLI_VERSION="+consts.CortexVersion)

	// Add a slight delay before running the command to ensure logs don't start until after the container is attached
	containerConfig.Cmd[0] = "sleep 0.1 && /root/check_cortex_version.sh && " + containerConfig.Cmd[0]

	dockerClient, err := docker.GetDockerClient()
	if err != nil {
		return "", nil, err
	}

	pulledImage, err := docker.PullImage(containerConfig.Image, docker.NoAuth, docker.PrintDots)
	if err != nil {
		if strings.Contains(err.Error(), "auth") {
			err = errors.Append(err, fmt.Sprintf("\n\nif your manager image is stored in a private repository: run `docker login` (if you haven't already), download your image with `docker pull %s`, and try this command again)", containerConfig.Image))
		}
		return "", nil, err
	}

	if pulledImage && addNewLineAfterPull {
		fmt.Println()
	}

	containerInfo, err := dockerClient.ContainerCreate(context.Background(), containerConfig, nil, nil, "")
	if err != nil {
		return "", nil, docker.WrapDockerError(err)
	}

	removeContainer := func() {
		dockerClient.ContainerRemove(context.Background(), containerInfo.ID, dockertypes.ContainerRemoveOptions{
			RemoveVolumes: true,
			Force:         true,
		})
	}

	defer removeContainer()

	// Make sure to remove container immediately on ctrl+c
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	caughtCtrlC := false

	routines.RunWithPanicHandler(func() {
		<-c
		caughtCtrlC = true
		removeContainer()
		exit.Error(ErrorDockerCtrlC())
	}, false)

	for _, copyPath := range copyToPaths {
		err = docker.CopyToContainer(containerInfo.ID, copyPath.input, copyPath.containerPath)
		if err != nil {
			return "", nil, err
		}
	}

	err = dockerClient.ContainerStart(context.Background(), containerInfo.ID, dockertypes.ContainerStartOptions{})
	if err != nil {
		return "", nil, docker.WrapDockerError(err)
	}

	// Use ContainerAttach() since that allows logs to be streamed even if they don't end in new lines
	logsOutput, err := dockerClient.ContainerAttach(context.Background(), containerInfo.ID, dockertypes.ContainerAttachOptions{
		Stream: true,
		Stdout: true,
		Stderr: true,
	})
	if err != nil {
		return "", nil, docker.WrapDockerError(err)
	}
	defer logsOutput.Close()

	var outputBuffer bytes.Buffer
	tee := io.TeeReader(logsOutput.Reader, &outputBuffer)

	_, err = io.Copy(os.Stdout, tee)
	if err != nil && err != io.EOF {
		return "", nil, errors.WithStack(err)
	}

	output := strings.ReplaceAll(outputBuffer.String(), "\r\n", "\n")

	// Let the ctrl+c handler run its course
	if caughtCtrlC {
		time.Sleep(5 * time.Second)
	}

	info, err := dockerClient.ContainerInspect(context.Background(), containerInfo.ID)
	if err != nil {
		return "", nil, errors.WithStack(err)
	}

	if info.State.ExitCode == 0 {
		for _, copyPath := range copyFromPaths {
			err = docker.CopyFromContainer(containerInfo.ID, copyPath.containerPath, copyPath.localDir)
			if err != nil {
				return "", nil, err
			}
		}
	}

	if info.State.Running {
		return output, nil, nil
	}

	return output, &info.State.ExitCode, nil
}

func runManagerWithClusterConfig(entrypoint string, clusterConfig *clusterconfig.Config, awsClient *aws.Client, copyToPaths []dockerCopyToPath, copyFromPaths []dockerCopyFromPath) (string, *int, error) {
	clusterConfigBytes, err := yaml.Marshal(clusterConfig)
	if err != nil {
		return "", nil, errors.WithStack(err)
	}

	cachedClusterConfigPath := cachedClusterConfigPath(clusterConfig.ClusterName, clusterConfig.Region)
	if err := files.WriteFile(clusterConfigBytes, cachedClusterConfigPath); err != nil {
		return "", nil, err
	}

	containerClusterConfigPath := "/in/" + filepath.Base(cachedClusterConfigPath)
	copyToPaths = append(copyToPaths, dockerCopyToPath{
		input: &archive.Input{
			Files: []archive.FileInput{
				{
					Source: cachedClusterConfigPath,
					Dest:   containerClusterConfigPath,
				},
			},
		},
		containerPath: "/",
	})

	containerConfig := &container.Config{
		Image:        clusterConfig.ImageManager,
		Entrypoint:   []string{"/bin/bash", "-c"},
		Cmd:          []string{fmt.Sprintf("eval $(python /root/cluster_config_env.py %s) && %s", containerClusterConfigPath, entrypoint)},
		Tty:          true,
		AttachStdout: true,
		AttachStderr: true,
		Env: []string{
			"CORTEX_PROVIDER=" + "aws",
			"AWS_ACCESS_KEY_ID=" + *awsClient.AccessKeyID(),
			"AWS_SECRET_ACCESS_KEY=" + *awsClient.SecretAccessKey(),
			"CORTEX_TELEMETRY_DISABLE=" + os.Getenv("CORTEX_TELEMETRY_DISABLE"),
			"CORTEX_TELEMETRY_SENTRY_DSN=" + os.Getenv("CORTEX_TELEMETRY_SENTRY_DSN"),
			"CORTEX_TELEMETRY_SEGMENT_WRITE_KEY=" + os.Getenv("CORTEX_TELEMETRY_SEGMENT_WRITE_KEY"),
			"CORTEX_DEV_DEFAULT_PREDICTOR_IMAGE_REGISTRY=" + os.Getenv("CORTEX_DEV_DEFAULT_PREDICTOR_IMAGE_REGISTRY_AWS"),
			"CORTEX_CLUSTER_CONFIG_FILE=" + containerClusterConfigPath,
		},
	}

	if sessionToken := awsClient.SessionToken(); sessionToken != nil {
		containerConfig.Env = append(containerConfig.Env, "AWS_SESSION_TOKEN="+*sessionToken)
	}

	output, exitCode, err := runManager(containerConfig, false, copyToPaths, copyFromPaths)
	if err != nil {
		return "", nil, err
	}

	return output, exitCode, nil
}

func runGCPManagerWithClusterConfig(entrypoint string, clusterConfig *clusterconfig.GCPConfig, copyToPaths []dockerCopyToPath, copyFromPaths []dockerCopyFromPath) (string, *int, error) {
	clusterConfigBytes, err := yaml.Marshal(clusterConfig)
	if err != nil {
		return "", nil, errors.WithStack(err)
	}

	clusterConfigPath := cachedGCPClusterConfigPath(clusterConfig.ClusterName, clusterConfig.Project, clusterConfig.Zone)
	if err := files.WriteFile(clusterConfigBytes, clusterConfigPath); err != nil {
		return "", nil, err
	}

	containerClusterConfigPath := "/in/" + filepath.Base(clusterConfigPath)
	gcpCredsPath := "/in/key.json"

	copyToPaths = append(copyToPaths,
		dockerCopyToPath{
			input: &archive.Input{
				Files: []archive.FileInput{
					{
						Source: clusterConfigPath,
						Dest:   containerClusterConfigPath,
					},
				},
			},
			containerPath: "/",
		},
		dockerCopyToPath{
			input: &archive.Input{
				Files: []archive.FileInput{
					{
						Source: os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"),
						Dest:   gcpCredsPath,
					},
				},
			},
			containerPath: "/",
		},
	)

	containerConfig := &container.Config{
		Image:        clusterConfig.ImageManager,
		Entrypoint:   []string{"/bin/bash", "-c"},
		Cmd:          []string{fmt.Sprintf("eval $(python /root/cluster_config_env.py %s) && %s", containerClusterConfigPath, entrypoint)},
		Tty:          true,
		AttachStdout: true,
		AttachStderr: true,
		Env: []string{
			"CORTEX_PROVIDER=gcp",
			"GOOGLE_APPLICATION_CREDENTIALS=" + gcpCredsPath,
			"CORTEX_GCP_PROJECT=" + clusterConfig.Project,
			"CORTEX_GCP_ZONE=" + clusterConfig.Zone,
			"CORTEX_TELEMETRY_DISABLE=" + os.Getenv("CORTEX_TELEMETRY_DISABLE"),
			"CORTEX_TELEMETRY_SENTRY_DSN=" + os.Getenv("CORTEX_TELEMETRY_SENTRY_DSN"),
			"CORTEX_TELEMETRY_SEGMENT_WRITE_KEY=" + os.Getenv("CORTEX_TELEMETRY_SEGMENT_WRITE_KEY"),
			"CORTEX_DEV_DEFAULT_PREDICTOR_IMAGE_REGISTRY=" + os.Getenv("CORTEX_DEV_DEFAULT_PREDICTOR_IMAGE_REGISTRY_GCP"),
			"CORTEX_CLUSTER_CONFIG_FILE=" + containerClusterConfigPath,
		},
	}

	output, exitCode, err := runManager(containerConfig, false, copyToPaths, copyFromPaths)
	if err != nil {
		return "", nil, err
	}

	return output, exitCode, nil
}

func runManagerAccessCommand(entrypoint string, accessConfig clusterconfig.AccessConfig, awsClient *aws.Client, copyToPaths []dockerCopyToPath, copyFromPaths []dockerCopyFromPath) (string, *int, error) {
	containerConfig := &container.Config{
		Image:        accessConfig.ImageManager,
		Entrypoint:   []string{"/bin/bash", "-c"},
		Cmd:          []string{entrypoint},
		Tty:          true,
		AttachStdout: true,
		AttachStderr: true,
		Env: []string{
			"CORTEX_PROVIDER=aws",
			"AWS_ACCESS_KEY_ID=" + *awsClient.AccessKeyID(),
			"AWS_SECRET_ACCESS_KEY=" + *awsClient.SecretAccessKey(),
			"CORTEX_CLUSTER_NAME=" + accessConfig.ClusterName,
			"CORTEX_REGION=" + accessConfig.Region,
			"CORTEX_TELEMETRY_DISABLE=" + os.Getenv("CORTEX_TELEMETRY_DISABLE"),
			"CORTEX_TELEMETRY_SENTRY_DSN=" + os.Getenv("CORTEX_TELEMETRY_SENTRY_DSN"),
			"CORTEX_TELEMETRY_SEGMENT_WRITE_KEY=" + os.Getenv("CORTEX_TELEMETRY_SEGMENT_WRITE_KEY"),
		},
	}

	if sessionToken := awsClient.SessionToken(); sessionToken != nil {
		containerConfig.Env = append(containerConfig.Env, "AWS_SESSION_TOKEN="+*sessionToken)
	}

	output, exitCode, err := runManager(containerConfig, true, copyToPaths, copyFromPaths)
	if err != nil {
		return "", nil, err
	}

	return output, exitCode, nil
}

func runGCPManagerAccessCommand(entrypoint string, accessConfig clusterconfig.GCPAccessConfig, copyToPaths []dockerCopyToPath, copyFromPaths []dockerCopyFromPath) (string, *int, error) {
	gcpCredsPath := "/in/key.json"

	copyToPaths = append(copyToPaths,
		dockerCopyToPath{
			input: &archive.Input{
				Files: []archive.FileInput{
					{
						Source: os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"),
						Dest:   gcpCredsPath,
					},
				},
			},
			containerPath: "/",
		},
	)

	containerConfig := &container.Config{
		Image:        accessConfig.ImageManager,
		Entrypoint:   []string{"/bin/bash", "-c"},
		Cmd:          []string{entrypoint},
		Tty:          true,
		AttachStdout: true,
		AttachStderr: true,
		Env: []string{
			"CORTEX_PROVIDER=gcp",
			"GOOGLE_APPLICATION_CREDENTIALS=" + gcpCredsPath,
			"CORTEX_CLUSTER_NAME=" + accessConfig.ClusterName,
			"CORTEX_GCP_PROJECT=" + accessConfig.Project,
			"CORTEX_GCP_ZONE=" + accessConfig.Zone,
			"CORTEX_TELEMETRY_DISABLE=" + os.Getenv("CORTEX_TELEMETRY_DISABLE"),
			"CORTEX_TELEMETRY_SENTRY_DSN=" + os.Getenv("CORTEX_TELEMETRY_SENTRY_DSN"),
			"CORTEX_TELEMETRY_SEGMENT_WRITE_KEY=" + os.Getenv("CORTEX_TELEMETRY_SEGMENT_WRITE_KEY"),
		},
	}

	output, exitCode, err := runManager(containerConfig, false, copyToPaths, copyFromPaths)
	if err != nil {
		return "", nil, err
	}

	return output, exitCode, nil
}
