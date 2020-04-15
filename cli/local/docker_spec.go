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
	"os"
	"path/filepath"

	"github.com/cortexlabs/cortex/cli/docker"
	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
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

func DeployContainers(api *spec.API) error {
	switch api.Predictor.Type {
	case userconfig.TensorFlowPredictorType:
		return TensorFlowSpec(api)
	case userconfig.ONNXPredictorType:
		return ONNXSpec(api)
	default:
		return PythonSpec(api)
	}
}

const (
	_specCacheDir                          = "/mnt/spec"
	_emptyDirMountPath                     = "/mnt"
	_emptyDirVolumeName                    = "mnt"
	_apiContainerName                      = "api"
	_tfServingContainerName                = "serve"
	_defaultPortInt32, _defaultPortStr     = int32(8888), "8888"
	_tfServingPortInt32, _tfServingPortStr = int32(9000), "9000"
	_requestMonitorReadinessFile           = "/request_monitor_ready.txt"
	_apiReadinessFile                      = "/mnt/api_readiness.txt"
	_apiLivenessFile                       = "/mnt/api_liveness.txt"
	_apiLivenessStalePeriod                = 7 // seconds (there is a 2-second buffer to be safe)
)

func getAPIEnv(api *spec.API) []string {
	envs := []string{}

	for envName, envVal := range api.Predictor.Env {
		envs = append(envs, fmt.Sprintf("%s=%s", envName, envVal))
	}

	envs = append(envs,
		"CORTEX_VERSION="+consts.CortexVersion,
		"CORTEX_SERVING_PORT="+_defaultPortStr,
		"CORTEX_PROVIDER="+"local",
		"CORTEX_CACHE_DIR=/mnt/cache",
		"CORTEX_MODEL_DIR=/mnt/model",
		"CORTEX_API_SPEC="+filepath.Join("/mnt/workspace", filepath.Base(api.Key)),
		"CORTEX_PROJECT_DIR="+"/mnt/project",
		"CORTEX_WORKERS_PER_REPLICA=1",
		"CORTEX_THREADS_PER_WORKER=1",
		"CORTEX_MAX_WORKER_CONCURRENCY=10",
		"CORTEX_SO_MAX_CONN=10",
	)

	if _, ok := api.Predictor.Env["PYTHONDONTWRITEBYTECODE"]; !ok {
		envs = append(envs, "PYTHONDONTWRITEBYTECODE=1")
	}
	return envs
}

func PythonSpec(api *spec.API) error {
	hostConfig := &container.HostConfig{
		PortBindings: nat.PortMap{
			"8888/tcp": []nat.PortBinding{
				{
					// HostPort: "8888",
				},
			},
		},
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: CWD,
				Target: "/mnt/project",
			},
			{
				Type:   mount.TypeBind,
				Source: filepath.Join(LocalWorkspace, filepath.Dir(api.Key)),
				Target: "/mnt/workspace",
			},
		},
	}

	containerConfig := &container.Config{
		Image: "cortexlabs/python-serve:latest",
		Env: append(
			getAPIEnv(api),
		),
		ExposedPorts: nat.PortSet{
			"8888/tcp": struct{}{},
		},
		Labels: map[string]string{
			"cortex":       "true",
			"type":         "api",
			"apiID":        api.ID,
			"apiName":      api.Name,
			"deploymentID": api.DeploymentID,
		},
	}
	containerInfo, err := docker.MustDockerClient().ContainerCreate(context.Background(), containerConfig, hostConfig, nil, "")
	if err != nil {
		return err
	}

	err = docker.MustDockerClient().ContainerStart(context.Background(), containerInfo.ID, dockertypes.ContainerStartOptions{})
	if err != nil {
		return err
	}

	return nil
}

func ONNXSpec(api *spec.API) error {
	hostConfig := &container.HostConfig{
		PortBindings: nat.PortMap{
			"8888/tcp": []nat.PortBinding{
				{
					// HostPort: "8888",
				},
			},
		},
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: CWD,
				Target: "/mnt/project",
			},
			{
				Type:   mount.TypeBind,
				Source: api.ModelMount.HostPath,
				Target: "/mnt/model",
			},
			{
				Type:   mount.TypeBind,
				Source: filepath.Join(LocalWorkspace, filepath.Dir(api.Key)),
				Target: "/mnt/workspace",
			},
		},
	}

	containerConfig := &container.Config{
		Image: "cortexlabs/onnx-serve:latest",
		Env: append(
			getAPIEnv(api),
			"CORTEX_VERSION=master",
		),
		ExposedPorts: nat.PortSet{
			"8888/tcp": struct{}{},
		},
		Labels: map[string]string{
			"cortex":       "true",
			"type":         _apiContainerName,
			"apiID":        api.ID,
			"apiName":      api.Name,
			"deploymentID": api.DeploymentID,
		},
	}
	containerInfo, err := docker.MustDockerClient().ContainerCreate(context.Background(), containerConfig, hostConfig, nil, "")
	if err != nil {
		return err
	}

	err = docker.MustDockerClient().ContainerStart(context.Background(), containerInfo.ID, dockertypes.ContainerStartOptions{})
	if err != nil {
		return err
	}

	return nil
}

func TensorFlowSpec(api *spec.API) error {
	serveHostConfig := &container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: api.ModelMount.HostPath,
				Target: "/mnt/model",
			},
		},
	}

	serveContainerConfig := &container.Config{
		Image: "cortexlabs/tf-serve:latest",
		Cmd: strslice.StrSlice{
			"--port=9000", "--model_base_path=/mnt/model",
		},
		ExposedPorts: nat.PortSet{
			"9000/tcp": struct{}{},
		},
		Labels: map[string]string{
			"cortex":       "true",
			"type":         _tfServingContainerName,
			"apiID":        api.ID,
			"apiName":      api.Name,
			"deploymentID": api.DeploymentID,
		},
	}

	containerCreateRequest, err := docker.MustDockerClient().ContainerCreate(context.Background(), serveContainerConfig, serveHostConfig, nil, "")
	if err != nil {
		return err
	}

	err = docker.MustDockerClient().ContainerStart(context.Background(), containerCreateRequest.ID, dockertypes.ContainerStartOptions{})
	if err != nil {
		return err
	}

	containerInfo, err := docker.MustDockerClient().ContainerInspect(context.Background(), containerCreateRequest.ID)
	tfContainerHost := containerInfo.NetworkSettings.Networks["bridge"].IPAddress

	apiHostConfig := &container.HostConfig{
		PortBindings: nat.PortMap{
			"8888/tcp": []nat.PortBinding{{}},
		},
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: CWD,
				Target: "/mnt/project",
			},
			{
				Type:   mount.TypeBind,
				Source: api.ModelMount.HostPath,
				Target: "/mnt/model",
			},
			{
				Type:   mount.TypeBind,
				Source: filepath.Join(LocalWorkspace, filepath.Dir(api.Key)),
				Target: "/mnt/workspace",
			},
		},
	}

	apiContainerConfig := &container.Config{
		Image: "cortexlabs/tf-api:latest",
		Env: append(
			getAPIEnv(api),
			"CORTEX_TF_SERVING_PORT="+"9000",
			"CORTEX_TF_SERVING_HOST="+tfContainerHost,
			"AWS_ACCESS_KEY_ID="+os.Getenv("AWS_ACCESS_KEY_ID"),
			"AWS_SECRET_ACCESS_KEY="+os.Getenv("AWS_SECRET_ACCESS_KEY"),
		),
		ExposedPorts: nat.PortSet{
			"8888/tcp": struct{}{},
		},
		Labels: map[string]string{
			"cortex":       "true",
			"type":         _apiContainerName,
			"apiID":        api.ID,
			"apiName":      api.Name,
			"deploymentID": api.DeploymentID,
		},
	}
	containerCreateRequest, err = docker.MustDockerClient().ContainerCreate(context.Background(), apiContainerConfig, apiHostConfig, nil, "")
	if err != nil {
		return err
	}

	err = docker.MustDockerClient().ContainerStart(context.Background(), containerCreateRequest.ID, dockertypes.ContainerStartOptions{})
	if err != nil {
		return err
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
		return nil, errors.WithStack(err)
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
		return err
	}
	return nil
}
