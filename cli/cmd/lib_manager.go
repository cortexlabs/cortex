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
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
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

func pullManager(clusterConfig *ClusterConfig) error {
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

func runManagerCommand(entrypoint string, clusterConfig *ClusterConfig) error {
	docker, err := getDockerClient()
	if err != nil {
		return err
	}

	pullManager(clusterConfig)

	containerConfig := &container.Config{
		Image:        clusterConfig.ImageManager,
		Entrypoint:   []string{entrypoint},
		Tty:          true,
		AttachStdout: true,
		AttachStderr: true,
		Env: []string{
			"AWS_ACCESS_KEY_ID=" + clusterConfig.AWSAccessKeyID,
			"AWS_SECRET_ACCESS_KEY=" + clusterConfig.AWSSecretAccessKey,
			"CORTEX_AWS_ACCESS_KEY_ID=" + clusterConfig.CortexAWSAccessKeyID,
			"CORTEX_AWS_SECRET_ACCESS_KEY=" + clusterConfig.CortexAWSSecretAccessKey,
			"CORTEX_INSTANCE_TYPE=" + *clusterConfig.InstanceType,
			"CORTEX_MIN_INSTANCES=" + s.Int64(*clusterConfig.MinInstances),
			"CORTEX_MAX_INSTANCES=" + s.Int64(*clusterConfig.MaxInstances),
			"CORTEX_CLUSTER_NAME=" + clusterConfig.ClusterName,
			"CORTEX_REGION=" + clusterConfig.Region,
			"CORTEX_BUCKET=" + clusterConfig.Bucket,
			"CORTEX_LOG_GROUP=" + clusterConfig.LogGroup,
			"CORTEX_TELEMETRY=" + s.Bool(clusterConfig.Telemetry),
			"CORTEX_IMAGE_FLUENTD=" + clusterConfig.ImageFluentd,
			"CORTEX_IMAGE_STATSD=" + clusterConfig.ImageStatsd,
			"CORTEX_IMAGE_OPERATOR=" + clusterConfig.ImageOperator,
			"CORTEX_IMAGE_TF_SERVE=" + clusterConfig.ImageTFServe,
			"CORTEX_IMAGE_TF_API=" + clusterConfig.ImageTFAPI,
			"CORTEX_IMAGE_TF_SERVE_GPU=" + clusterConfig.ImageTFServeGPU,
			"CORTEX_IMAGE_ONNX_SERVE=" + clusterConfig.ImageOnnxServe,
			"CORTEX_IMAGE_ONNX_SERVE_GPU=" + clusterConfig.ImageOnnxServeGPU,
			"CORTEX_IMAGE_CLUSTER_AUTOSCALER=" + clusterConfig.ImageClusterAutoscaler,
			"CORTEX_IMAGE_NVIDIA=" + clusterConfig.ImageNvidia,
			"CORTEX_IMAGE_METRICS_SERVER=" + clusterConfig.ImageMetricsServer,
			"CORTEX_IMAGE_ISTIO_CITADEL=" + clusterConfig.ImageIstioCitadel,
			"CORTEX_IMAGE_ISTIO_GALLEY=" + clusterConfig.ImageIstioGalley,
			"CORTEX_IMAGE_ISTIO_PILOT=" + clusterConfig.ImageIstioPilot,
			"CORTEX_IMAGE_ISTIO_PROXY=" + clusterConfig.ImageIstioProxy,
			"CORTEX_IMAGE_DOWNLOADER=" + clusterConfig.ImageDownloader,
		},
	}
	containerInfo, err := docker.ContainerCreate(context.Background(), containerConfig, nil, nil, "")
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

func installEKS(clusterConfig *ClusterConfig) error {
	err := runManagerCommand("/root/install_eks.sh", clusterConfig)
	if err != nil {
		return err
	}

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
