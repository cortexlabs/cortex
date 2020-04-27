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

package local

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/docker"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"github.com/docker/docker/api/types"
	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/go-connections/nat"
)

const (
	_apiContainerName       = "api"
	_tfServingContainerName = "serve"
	_defaultPortStr         = "8888"
	_tfServingPortStr       = "9000"
	_projectDir             = "/mnt/project"
	_cacheDir               = "/mnt/cache"
	_modelDir               = "/mnt/model"
	_workspaceDir           = "/mnt/workspace"
)

func DeployContainers(api *spec.API, awsClient *aws.Client) error {
	switch api.Predictor.Type {
	case userconfig.TensorFlowPredictorType:
		return deployTensorFlowContainers(api, awsClient)
	case userconfig.ONNXPredictorType:
		return deployONNXContainer(api, awsClient)
	default:
		return deployPythonContainer(api, awsClient)
	}
}

func getAPIEnv(api *spec.API, awsClient *aws.Client) []string {
	envs := []string{}

	for envName, envVal := range api.Predictor.Env {
		envs = append(envs, fmt.Sprintf("%s=%s", envName, envVal))
	}

	envs = append(envs,
		"CORTEX_VERSION="+consts.CortexVersion,
		"CORTEX_SERVING_PORT="+_defaultPortStr,
		"CORTEX_PROVIDER="+"local",
		"CORTEX_CACHE_DIR="+_cacheDir,
		"CORTEX_MODEL_DIR="+_modelDir,
		"CORTEX_API_SPEC="+filepath.Join("/mnt/workspace", filepath.Base(api.Key)),
		"CORTEX_PROJECT_DIR="+_projectDir,
		"CORTEX_WORKERS_PER_REPLICA=1",
		"CORTEX_THREADS_PER_WORKER=1",
		"CORTEX_MAX_WORKER_CONCURRENCY=1000",
		"CORTEX_SO_MAX_CONN=1000",
		"AWS_REGION="+awsClient.Region,
	)

	if awsAccessKeyID := awsClient.AccessKeyID(); awsAccessKeyID != nil {
		envs = append(envs, "AWS_ACCESS_KEY_ID="+*awsAccessKeyID)
	}

	if awsSecretAccessKey := awsClient.SecretAccessKey(); awsSecretAccessKey != nil {
		envs = append(envs, "AWS_SECRET_ACCESS_KEY="+*awsSecretAccessKey)
	}

	if _, ok := api.Predictor.Env["PYTHONDONTWRITEBYTECODE"]; !ok {
		envs = append(envs, "PYTHONDONTWRITEBYTECODE=1")
	}
	return envs
}

func deployPythonContainer(api *spec.API, awsClient *aws.Client) error {
	portBinding := nat.PortBinding{}
	if api.LocalPort != nil {
		portBinding.HostPort = s.Int(*api.LocalPort)
	}

	runtime := ""
	resources := container.Resources{}
	if api.Compute != nil {
		if api.Compute.CPU != nil {
			resources.NanoCPUs = api.Compute.CPU.MilliValue() * 1000 * 1000
		}
		if api.Compute.Mem != nil {
			resources.Memory = api.Compute.Mem.Quantity.Value()
		}
		if api.Compute.GPU > 0 {
			runtime = "nvidia"
		}
	}

	hostConfig := &container.HostConfig{
		PortBindings: nat.PortMap{
			_defaultPortStr + "/tcp": []nat.PortBinding{portBinding},
		},
		Runtime:   runtime,
		Resources: resources,
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: api.LocalProjectDir,
				Target: _projectDir,
			},
			{
				Type:   mount.TypeBind,
				Source: filepath.Join(_localWorkspaceDir, filepath.Dir(api.Key)),
				Target: "/mnt/workspace",
			},
		},
	}

	containerConfig := &container.Config{
		Image: api.Predictor.Image,
		Env: append(
			getAPIEnv(api, awsClient),
		),
		ExposedPorts: nat.PortSet{
			_defaultPortStr + "/tcp": struct{}{},
		},
		Labels: map[string]string{
			"cortex":  "true",
			"type":    _apiContainerName,
			"apiID":   api.ID,
			"apiName": api.Name,
		},
	}
	containerInfo, err := docker.MustDockerClient().ContainerCreate(context.Background(), containerConfig, hostConfig, nil, "")
	if err != nil {
		return errors.Wrap(err, "api", api.Identify())
	}

	err = docker.MustDockerClient().ContainerStart(context.Background(), containerInfo.ID, dockertypes.ContainerStartOptions{})
	if err != nil {
		return errors.Wrap(err, "api", api.Identify())
	}

	return nil
}

func deployONNXContainer(api *spec.API, awsClient *aws.Client) error {
	portBinding := nat.PortBinding{}
	if api.LocalPort != nil {
		portBinding.HostPort = s.Int(*api.LocalPort)
	}

	runtime := ""
	resources := container.Resources{}
	if api.Compute != nil {
		if api.Compute.CPU != nil {
			resources.NanoCPUs = api.Compute.CPU.MilliValue() * 1000 * 1000
		}
		if api.Compute.Mem != nil {
			resources.Memory = api.Compute.Mem.Quantity.Value()
		}
		if api.Compute.GPU > 0 {
			runtime = "nvidia"
		}
	}

	hostConfig := &container.HostConfig{
		PortBindings: nat.PortMap{
			_defaultPortStr + "/tcp": []nat.PortBinding{portBinding},
		},
		Runtime:   runtime,
		Resources: resources,
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: api.LocalProjectDir,
				Target: _projectDir,
			},
			{
				Type:   mount.TypeBind,
				Source: filepath.Join(_localWorkspaceDir, filepath.Dir(api.Key)),
				Target: "/mnt/workspace",
			},
			{
				Type:   mount.TypeBind,
				Source: api.LocalModelCache.HostPath,
				Target: _modelDir,
			},
		},
	}

	containerConfig := &container.Config{
		Image: api.Predictor.Image,
		Env: append(
			getAPIEnv(api, awsClient),
		),
		ExposedPorts: nat.PortSet{
			_defaultPortStr + "/tcp": struct{}{},
		},
		Labels: map[string]string{
			"cortex":  "true",
			"type":    _apiContainerName,
			"apiID":   api.ID,
			"apiName": api.Name,
			"modelID": api.LocalModelCache.ID,
		},
	}
	containerInfo, err := docker.MustDockerClient().ContainerCreate(context.Background(), containerConfig, hostConfig, nil, "")
	if err != nil {
		return errors.Wrap(err, "api", api.Identify())
	}

	err = docker.MustDockerClient().ContainerStart(context.Background(), containerInfo.ID, dockertypes.ContainerStartOptions{})
	if err != nil {
		return errors.Wrap(err, "api", api.Identify())
	}

	return nil
}

func deployTensorFlowContainers(api *spec.API, awsClient *aws.Client) error {
	serveRuntime := ""
	serveResources := container.Resources{}
	apiResources := container.Resources{}

	if api.Compute != nil {
		if api.Compute.CPU != nil {
			totalNanoCPUs := api.Compute.CPU.MilliValue() * 1000 * 1000
			apiResources.NanoCPUs = totalNanoCPUs / 2
			serveResources.NanoCPUs = totalNanoCPUs - apiResources.NanoCPUs
		}
		if api.Compute.Mem != nil {
			totalMemory := api.Compute.Mem.Quantity.Value()
			apiResources.Memory = totalMemory / 2
			serveResources.Memory = totalMemory - apiResources.Memory
		}
		if api.Compute.GPU > 0 {
			serveRuntime = "nvidia"
		}
	}

	serveHostConfig := &container.HostConfig{
		Runtime:   serveRuntime,
		Resources: serveResources,
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: api.LocalModelCache.HostPath,
				Target: _modelDir,
			},
		},
	}

	serveContainerConfig := &container.Config{
		Image: api.Predictor.TFServeImage,
		Cmd: strslice.StrSlice{
			"--port=" + _tfServingPortStr, "--model_base_path=" + _modelDir,
		},
		ExposedPorts: nat.PortSet{
			_tfServingPortStr + "/tcp": struct{}{},
		},
		Labels: map[string]string{
			"cortex":  "true",
			"type":    _tfServingContainerName,
			"apiID":   api.ID,
			"apiName": api.Name,
			"modelID": api.LocalModelCache.ID,
		},
	}

	containerCreateRequest, err := docker.MustDockerClient().ContainerCreate(context.Background(), serveContainerConfig, serveHostConfig, nil, "")
	if err != nil {
		return errors.Wrap(err, "api", api.Identify())
	}

	err = docker.MustDockerClient().ContainerStart(context.Background(), containerCreateRequest.ID, dockertypes.ContainerStartOptions{})
	if err != nil {
		return errors.Wrap(err, "api", api.Identify())
	}

	containerInfo, err := docker.MustDockerClient().ContainerInspect(context.Background(), containerCreateRequest.ID)
	if err != nil {
		return errors.Wrap(err, "api", api.Identify())
	}

	tfContainerHost := containerInfo.NetworkSettings.Networks["bridge"].IPAddress

	portBinding := nat.PortBinding{}
	if api.LocalPort != nil {
		portBinding.HostPort = fmt.Sprintf("%d", *api.LocalPort)
	}
	apiHostConfig := &container.HostConfig{
		PortBindings: nat.PortMap{
			_defaultPortStr + "/tcp": []nat.PortBinding{portBinding},
		},
		Resources: apiResources,
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: _cwd,
				Target: _projectDir,
			},
			{
				Type:   mount.TypeBind,
				Source: api.LocalModelCache.HostPath,
				Target: _modelDir,
			},
			{
				Type:   mount.TypeBind,
				Source: filepath.Join(_localWorkspaceDir, filepath.Dir(api.Key)),
				Target: "/mnt/workspace",
			},
		},
	}

	apiContainerConfig := &container.Config{
		Image: api.Predictor.Image,
		Env: append(
			getAPIEnv(api, awsClient),
			"CORTEX_TF_SERVING_PORT="+_tfServingPortStr,
			"CORTEX_TF_SERVING_HOST="+tfContainerHost,
		),
		ExposedPorts: nat.PortSet{
			_defaultPortStr + "/tcp": struct{}{},
		},
		Labels: map[string]string{
			"cortex":  "true",
			"type":    _apiContainerName,
			"apiID":   api.ID,
			"apiName": api.Name,
			"modelID": api.LocalModelCache.ID,
		},
	}
	containerCreateRequest, err = docker.MustDockerClient().ContainerCreate(context.Background(), apiContainerConfig, apiHostConfig, nil, "")
	if err != nil {
		return errors.Wrap(err, "api", api.Identify())
	}

	err = docker.MustDockerClient().ContainerStart(context.Background(), containerCreateRequest.ID, dockertypes.ContainerStartOptions{})
	if err != nil {
		return errors.Wrap(err, "api", api.Identify())
	}

	return nil
}

func GetContainersByAPI(apiName string) ([]dockertypes.Container, error) {
	dargs := filters.NewArgs()
	dargs.Add("label", "cortex=true")
	dargs.Add("label", "apiName="+apiName)

	containers, err := docker.MustDockerClient().ContainerList(context.Background(), types.ContainerListOptions{
		All:     true,
		Filters: dargs,
	})
	if err != nil {
		return nil, errors.Wrap(err, "api", apiName)
	}

	return containers, nil
}

func DeleteContainers(apiName string) error {
	containers, err := GetContainersByAPI(apiName)
	if err != nil {
		return err
	}

	for _, container := range containers {
		attemptErr := docker.MustDockerClient().ContainerRemove(context.Background(), container.ID, dockertypes.ContainerRemoveOptions{
			RemoveVolumes: true,
			Force:         true,
		})
		if attemptErr != nil {
			err = attemptErr
		}
	}
	if err != nil {
		return errors.Wrap(err, "api", apiName)
	}
	return nil
}
