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
	"strings"

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
	"github.com/docker/go-connections/nat"
)

const (
	_apiContainerName          = "api"
	_tfServingContainerName    = "serve"
	_defaultPortStr            = "8888"
	_tfServingPortStr          = "9000"
	_tfServingEmptyModelConfig = "/etc/tfs/model_config_server.conf"
	_tfServingBatchConfig      = "/etc/tfs/batch_config.conf"
	_projectDir                = "/mnt/project"
	_cacheDir                  = "/mnt/cache"
	_modelDir                  = "/mnt/model"
	_workspaceDir              = "/mnt/workspace"
)

type ModelCaches []*spec.LocalModelCache

func (modelCaches ModelCaches) IDs() string {
	ids := make([]string, len(modelCaches))
	for i, modelCache := range modelCaches {
		ids[i] = modelCache.ID
	}

	return strings.Join(ids, ", ")
}

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
		"CORTEX_KIND="+api.Kind.String(),
		"CORTEX_VERSION="+consts.CortexVersion,
		"CORTEX_SERVING_PORT="+_defaultPortStr,
		"CORTEX_PROVIDER="+"local",
		"CORTEX_CACHE_DIR="+_cacheDir,
		"CORTEX_MODEL_DIR="+_modelDir,
		"CORTEX_MODELS="+strings.Join(api.ModelNames(), ","),
		"CORTEX_API_SPEC="+filepath.Join("/mnt/workspace", filepath.Base(api.Key)),
		"CORTEX_PROJECT_DIR="+_projectDir,
		"CORTEX_PROCESSES_PER_REPLICA="+s.Int32(api.Predictor.ProcessesPerReplica),
		"CORTEX_THREADS_PER_PROCESS="+s.Int32(api.Predictor.ThreadsPerProcess),
		"CORTEX_MAX_REPLICA_CONCURRENCY="+s.Int32(api.Predictor.ProcessesPerReplica*api.Predictor.ThreadsPerProcess),
		// add 1 because it was required to achieve the target concurrency for 1 process, 1 thread
		"AWS_REGION="+awsClient.Region,
	)

	cortexPythonPath := _projectDir
	if api.Predictor.PythonPath != nil {
		cortexPythonPath = filepath.Join(_projectDir, *api.Predictor.PythonPath)
	}
	envs = append(envs, "CORTEX_PYTHON_PATH="+cortexPythonPath)

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
	if api.Networking.LocalPort != nil {
		portBinding.HostPort = s.Int(*api.Networking.LocalPort)
	}

	resources := container.Resources{}
	if api.Compute != nil {
		if api.Compute.CPU != nil {
			resources.NanoCPUs = api.Compute.CPU.MilliValue() * 1000 * 1000
		}
		if api.Compute.Mem != nil {
			resources.Memory = api.Compute.Mem.Quantity.Value()
		}
		if api.Compute.GPU > 0 {
			resources.DeviceRequests = []container.DeviceRequest{{
				Count: -1,
				Capabilities: [][]string{
					{"gpu"},
				},
			}}
		}
	}

	hostConfig := &container.HostConfig{
		PortBindings: nat.PortMap{
			_defaultPortStr + "/tcp": []nat.PortBinding{portBinding},
		},
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
				Target: _workspaceDir,
			},
		},
	}

	containerConfig := &container.Config{
		Image: api.Predictor.Image,
		Tty:   true,
		Env: append(
			getAPIEnv(api, awsClient),
		),
		ExposedPorts: nat.PortSet{
			_defaultPortStr + "/tcp": struct{}{},
		},
		Labels: map[string]string{
			"cortex":      "true",
			"type":        _apiContainerName,
			"apiID":       api.ID,
			"specID":      api.SpecID,
			"predictorID": api.PredictorID,
			"apiName":     api.Name,
		},
	}
	containerInfo, err := docker.MustDockerClient().ContainerCreate(context.Background(), containerConfig, hostConfig, nil, "")
	if err != nil {
		if strings.Contains(err.Error(), "bind source path does not exist") {
			return errors.Wrap(ErrorBindDockerInDocker(err), api.Identify())
		}
		return errors.Wrap(err, api.Identify())
	}

	err = docker.MustDockerClient().ContainerStart(context.Background(), containerInfo.ID, dockertypes.ContainerStartOptions{})
	if err != nil {
		if api.Compute.GPU == 0 {
			return errors.Wrap(err, api.Identify())
		}
		err := retryWithNvidiaRuntime(err, containerConfig, hostConfig)
		if err != nil {
			return errors.Wrap(err, api.Identify())
		}
	}

	return nil
}

func deployONNXContainer(api *spec.API, awsClient *aws.Client) error {
	portBinding := nat.PortBinding{}
	if api.Networking.LocalPort != nil {
		portBinding.HostPort = s.Int(*api.Networking.LocalPort)
	}

	resources := container.Resources{}
	if api.Compute != nil {
		if api.Compute.CPU != nil {
			resources.NanoCPUs = api.Compute.CPU.MilliValue() * 1000 * 1000
		}
		if api.Compute.Mem != nil {
			resources.Memory = api.Compute.Mem.Quantity.Value()
		}
		if api.Compute.GPU > 0 {
			resources.DeviceRequests = []container.DeviceRequest{{
				Count: -1,
				Capabilities: [][]string{
					{"gpu"},
				},
			}}
		}
	}

	mounts := []mount.Mount{
		{
			Type:   mount.TypeBind,
			Source: api.LocalProjectDir,
			Target: _projectDir,
		},
		{
			Type:   mount.TypeBind,
			Source: filepath.Join(_localWorkspaceDir, filepath.Dir(api.Key)),
			Target: _workspaceDir,
		},
	}
	for _, modelCache := range api.LocalModelCaches {
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: modelCache.HostPath,
			Target: filepath.Join(_modelDir, modelCache.TargetPath),
		})
	}

	hostConfig := &container.HostConfig{
		PortBindings: nat.PortMap{
			_defaultPortStr + "/tcp": []nat.PortBinding{portBinding},
		},
		Resources: resources,
		Mounts:    mounts,
	}

	containerConfig := &container.Config{
		Image: api.Predictor.Image,
		Tty:   true,
		Env: append(
			getAPIEnv(api, awsClient),
		),
		ExposedPorts: nat.PortSet{
			_defaultPortStr + "/tcp": struct{}{},
		},
		Labels: map[string]string{
			"cortex":      "true",
			"type":        _apiContainerName,
			"apiID":       api.ID,
			"specID":      api.SpecID,
			"predictorID": api.PredictorID,
			"apiName":     api.Name,
			"modelIDs":    ModelCaches(api.LocalModelCaches).IDs(),
		},
	}
	containerInfo, err := docker.MustDockerClient().ContainerCreate(context.Background(), containerConfig, hostConfig, nil, "")
	if err != nil {
		if strings.Contains(err.Error(), "bind source path does not exist") {
			return errors.Wrap(ErrorBindDockerInDocker(err), api.Identify())
		}
		return errors.Wrap(err, api.Identify())
	}

	err = docker.MustDockerClient().ContainerStart(context.Background(), containerInfo.ID, dockertypes.ContainerStartOptions{})
	if err != nil {
		if api.Compute.GPU == 0 {
			return errors.Wrap(err, api.Identify())
		}
		err := retryWithNvidiaRuntime(err, containerConfig, hostConfig)
		if err != nil {
			return errors.Wrap(err, api.Identify())
		}
	}

	return nil
}

func deployTensorFlowContainers(api *spec.API, awsClient *aws.Client) error {
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
			serveResources.DeviceRequests = append(serveResources.DeviceRequests, container.DeviceRequest{
				Count: -1,
				Capabilities: [][]string{
					{"gpu"},
				},
			})
		}
	}

	mounts := []mount.Mount{}
	for _, modelCache := range api.LocalModelCaches {
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: modelCache.HostPath,
			Target: filepath.Join(_modelDir, modelCache.TargetPath),
		})
	}

	serveHostConfig := &container.HostConfig{
		Resources: serveResources,
		Mounts:    mounts,
	}

	envVars := []string{}
	cmdArgs := []string{
		"--port=" + _tfServingPortStr,
		"--model_config_file=" + _tfServingEmptyModelConfig,
	}
	if api.Predictor.ServerSideBatching != nil {
		envVars = append(envVars,
			"TF_MAX_BATCH_SIZE="+s.Int32(api.Predictor.ServerSideBatching.MaxBatchSize),
			"TF_BATCH_TIMEOUT_MICROS="+s.Int64(api.Predictor.ServerSideBatching.BatchInterval.Microseconds()),
			"TF_NUM_BATCHED_THREADS="+s.Int32(api.Predictor.ProcessesPerReplica),
		)
		cmdArgs = append(cmdArgs,
			"--enable_batching=true",
			"--batching_parameters_file="+_tfServingBatchConfig,
		)
	}

	serveContainerConfig := &container.Config{
		Image: api.Predictor.TensorFlowServingImage,
		Tty:   true,
		Env:   envVars,
		Cmd:   cmdArgs,
		ExposedPorts: nat.PortSet{
			_tfServingPortStr + "/tcp": struct{}{},
		},
		Labels: map[string]string{
			"cortex":      "true",
			"type":        _tfServingContainerName,
			"apiID":       api.ID,
			"specID":      api.SpecID,
			"predictorID": api.PredictorID,
			"apiName":     api.Name,
			"modelIDs":    ModelCaches(api.LocalModelCaches).IDs(),
		},
	}

	containerCreateRequest, err := docker.MustDockerClient().ContainerCreate(context.Background(), serveContainerConfig, serveHostConfig, nil, "")
	if err != nil {
		if strings.Contains(err.Error(), "bind source path does not exist") {
			return errors.Wrap(ErrorBindDockerInDocker(err), api.Identify())
		}
		return errors.Wrap(err, api.Identify())
	}

	err = docker.MustDockerClient().ContainerStart(context.Background(), containerCreateRequest.ID, dockertypes.ContainerStartOptions{})
	if err != nil {
		if api.Compute.GPU == 0 {
			return errors.Wrap(err, api.Identify())
		}
		err := retryWithNvidiaRuntime(err, serveContainerConfig, serveHostConfig)
		if err != nil {
			return errors.Wrap(err, api.Identify())
		}
	}

	containerInfo, err := docker.MustDockerClient().ContainerInspect(context.Background(), containerCreateRequest.ID)
	if err != nil {
		return errors.Wrap(err, api.Identify())
	}

	tfContainerHost := containerInfo.NetworkSettings.Networks["bridge"].IPAddress

	portBinding := nat.PortBinding{}
	if api.Networking.LocalPort != nil {
		portBinding.HostPort = fmt.Sprintf("%d", *api.Networking.LocalPort)
	}
	apiHostConfig := &container.HostConfig{
		PortBindings: nat.PortMap{
			_defaultPortStr + "/tcp": []nat.PortBinding{portBinding},
		},
		Resources: apiResources,
		Mounts: append([]mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: api.LocalProjectDir,
				Target: _projectDir,
			},
			{
				Type:   mount.TypeBind,
				Source: filepath.Join(_localWorkspaceDir, filepath.Dir(api.Key)),
				Target: _workspaceDir,
			},
		}, mounts...),
	}

	apiContainerConfig := &container.Config{
		Image: api.Predictor.Image,
		Tty:   true,
		Env: append(
			getAPIEnv(api, awsClient),
			"CORTEX_TF_BASE_SERVING_PORT="+_tfServingPortStr,
			"CORTEX_TF_SERVING_HOST="+tfContainerHost,
		),
		ExposedPorts: nat.PortSet{
			_defaultPortStr + "/tcp": struct{}{},
		},
		Labels: map[string]string{
			"cortex":      "true",
			"type":        _apiContainerName,
			"apiID":       api.ID,
			"specID":      api.SpecID,
			"predictorID": api.PredictorID,
			"apiName":     api.Name,
			"modelIDs":    ModelCaches(api.LocalModelCaches).IDs(),
		},
	}
	containerCreateRequest, err = docker.MustDockerClient().ContainerCreate(context.Background(), apiContainerConfig, apiHostConfig, nil, "")
	if err != nil {
		if strings.Contains(err.Error(), "bind source path does not exist") {
			return errors.Wrap(ErrorBindDockerInDocker(err), api.Identify())
		}
		return errors.Wrap(err, api.Identify())
	}

	err = docker.MustDockerClient().ContainerStart(context.Background(), containerCreateRequest.ID, dockertypes.ContainerStartOptions{})
	if err != nil {
		return errors.Wrap(err, api.Identify())
	}

	return nil
}

// Retries deploying a container requiring GPU using nvidia runtime, returns original error if isn't relevant, nil if successful and new error if a retry was attempted but failed
func retryWithNvidiaRuntime(err error, containerConfig *container.Config, hostConfig *container.HostConfig) error {
	// error message if device driver may look like 'could not select device driver "" with capabilities: [[gpu]]'
	if !(strings.Contains(err.Error(), "could not select device driver") && strings.Contains(err.Error(), "gpu")) {
		return err
	}

	if _, ok := docker.MustDockerClient().Info.Runtimes["nvidia"]; ok {
		localPrintln("retrying API deployment using nvidia runtime because device driver for GPU was not found")
		hostConfig.Runtime = "nvidia"
		hostConfig.Resources.DeviceRequests = nil
		containerCreateRequest, err := docker.MustDockerClient().ContainerCreate(context.Background(), containerConfig, hostConfig, nil, "")
		if err != nil {
			return errors.Wrap(err, "failed to request a GPU")
		}
		err = docker.MustDockerClient().ContainerStart(context.Background(), containerCreateRequest.ID, dockertypes.ContainerStartOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to run a container using nvidia runtime; it is recommended to use the latest Docker Engine (https://docs.docker.com/engine/install/) with nvidia-container-runtime or nvidia-container-toolkit (https://docs.docker.com/config/containers/resource_constraints/#gpu)")
		}
		return nil
	}
	return errors.Append(errors.Wrap(err, "failed to allocate GPU"), "\n\n* only NVIDIA gpus are supported\n* please make sure that you've set up nvidia-container-runtime or nvidia-container-toolkit for your Docker Engine correctly (https://docs.docker.com/config/containers/resource_constraints/#gpu)\n\nAlternatively, try deploying the API without requesting a GPU by updating `compute.gpu` in your API configuration yaml")
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

func GetAllRunningContainers() ([]dockertypes.Container, error) {
	dargs := filters.NewArgs()
	dargs.Add("label", "cortex=true")

	containers, err := docker.MustDockerClient().ContainerList(context.Background(), types.ContainerListOptions{
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
		return errors.Wrap(err, "api", apiName)
	}
	return nil
}
