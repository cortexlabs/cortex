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
	"strings"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/hash"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/msgpack"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/zip"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
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

func Deploy(configPath string, deploymentBytesMap map[string][]byte) (schema.DeployResponse, error) {
	projectFileMap, err := zip.UnzipMemToMem(deploymentBytesMap["project.zip"])
	if err != nil {
		return schema.DeployResponse{}, err
	}

	apiConfigs, err := spec.ExtractAPIConfigs(deploymentBytesMap["config"], projectFileMap, configPath)
	if err != nil {
		return schema.DeployResponse{}, err
	}

	err = ValidateLocalAPIs(apiConfigs, projectFileMap)
	if err != nil {
		return schema.DeployResponse{}, err
	}
	projectID := hash.Bytes(deploymentBytesMap["project.zip"])

	// TODO try to pickup AWS credentials silently if aws creds in local environment are empty

	// TODO use credentials from Local environment
	os.Setenv("AWS_ACCESS_KEY_ID", "")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "")

	results := make([]schema.DeployResult, len(apiConfigs))
	for i, apiConfig := range apiConfigs {
		if apiConfig.Predictor.Model != nil {
			path, err := CacheModel(&apiConfig)
			if err != nil {
				results[i].Error = errors.Message(errors.Wrap(err, apiConfig.Name, userconfig.PredictorKey, userconfig.ModelKey))
			}
			apiConfig.Predictor.Model = pointer.String(path)
		}
		api, msg, err := UpdateAPI(&apiConfig, projectID)
		results[i].Message = msg
		if err != nil {
			results[i].Error = errors.Message(err)
		} else {
			results[i].API = *api
		}
	}

	return schema.DeployResponse{
		Results: results,
	}, nil
}

func GetContainerByAPI(apiName string) ([]dockertypes.Container, error) {
	dargs := filters.NewArgs()
	dargs.Add("label", "cortex=true")
	dargs.Add("label", "apiName="+apiName)

	containers, err := DockerClient().ContainerList(context.Background(), types.ContainerListOptions{
		All:     true,
		Filters: dargs,
	})
	if err != nil {
		return nil, err
	}

	return containers, nil
}

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

func getAPIEnv(api *spec.API) []string {
	envs := []string{}

	for envName, envVal := range api.Predictor.Env {
		envs = append(envs, fmt.Sprintf("%s=%s", envName, envVal))
	}

	return envs
}

func PythonSpec(api *spec.API) error {
	apiBytes, err := msgpack.Marshal(api)
	if err != nil {
		exit.Error(err)
	}

	os.MkdirAll(files.ParentDir(filepath.Join(LocalWorkspace, api.Key)), os.ModePerm)
	err = files.WriteFile(apiBytes, filepath.Join(LocalWorkspace, api.Key))

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
		Image:        "cortexlabs/python-serve:latest",
		Tty:          true,
		AttachStdout: true,
		AttachStderr: true,
		Env: append(
			getAPIEnv(api),
			"CORTEX_VERSION=master",
			"CORTEX_SERVING_PORT=8888",
			"CORTEX_PROVIDER=local",
			"CORTEX_CACHE_DIR="+"/mnt/cache",
			"CORTEX_API_SPEC="+filepath.Join("/mnt/workspace", filepath.Base(api.Key)),
			"CORTEX_PROJECT_DIR="+"/mnt/project",
			"CORTEX_WORKERS_PER_REPLICA=1",
			"CORTEX_MAX_WORKER_CONCURRENCY=10",
			"CORTEX_SO_MAX_CONN=10",
			"CORTEX_THREADS_PER_WORKER=1",
			"AWS_ACCESS_KEY_ID="+os.Getenv("AWS_ACCESS_KEY_ID"),
			"AWS_SECRET_ACCESS_KEY="+os.Getenv("AWS_SECRET_ACCESS_KEY"),
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
	containerInfo, err := DockerClient().ContainerCreate(context.Background(), containerConfig, hostConfig, nil, "")
	if err != nil {
		return err
	}

	err = DockerClient().ContainerStart(context.Background(), containerInfo.ID, dockertypes.ContainerStartOptions{})
	if err != nil {
		return err
	}

	return nil
}

func ONNXSpec(api *spec.API) error {
	modelPath := *api.Predictor.Model

	mountedModelPath := filepath.Join("/mnt/model", filepath.Base(modelPath))

	api.Predictor.Model = pointer.String(mountedModelPath)

	apiBytes, err := msgpack.Marshal(api)
	if err != nil {
		exit.Error(err)
	}

	os.MkdirAll(files.ParentDir(filepath.Join(LocalWorkspace, api.Key)), os.ModePerm)
	err = files.WriteFile(apiBytes, filepath.Join(LocalWorkspace, api.Key))

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
				Source: filepath.Dir(modelPath),
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
		Image:        "cortexlabs/onnx-serve:latest",
		Tty:          true,
		AttachStdout: true,
		AttachStderr: true,
		Env: append(
			getAPIEnv(api),
			"CORTEX_VERSION=master",
			"CORTEX_SERVING_PORT=8888",
			"CORTEX_PROVIDER=local",
			"CORTEX_CACHE_DIR="+"/mnt/cache",
			"CORTEX_API_SPEC="+filepath.Join("/mnt/workspace", api.Key),
			"CORTEX_PROJECT_DIR="+"/mnt/project",
			"CORTEX_WORKERS_PER_REPLICA=1",
			"CORTEX_MAX_WORKER_CONCURRENCY=10",
			"CORTEX_SO_MAX_CONN=10",
			"CORTEX_THREADS_PER_WORKER=1",
			"AWS_ACCESS_KEY_ID="+os.Getenv("AWS_ACCESS_KEY_ID"),
			"AWS_SECRET_ACCESS_KEY="+os.Getenv("AWS_SECRET_ACCESS_KEY"),
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
	containerInfo, err := DockerClient().ContainerCreate(context.Background(), containerConfig, hostConfig, nil, "")
	if err != nil {
		return err
	}

	err = DockerClient().ContainerStart(context.Background(), containerInfo.ID, dockertypes.ContainerStartOptions{})
	if err != nil {
		return err
	}

	return nil
}

func TensorFlowSpec(api *spec.API) error {
	modelPath := *api.Predictor.Model

	mountPath := modelPath
	if strings.HasSuffix(modelPath, ".zip") {
		mountPath = filepath.Dir(modelPath)
		fmt.Println(mountPath)
		_, err := zip.UnzipFileToDir(modelPath, mountPath)
		if err != nil {
			return err
		}
	}

	api.Predictor.Model = pointer.String("/mnt/model")

	apiBytes, err := msgpack.Marshal(api)
	if err != nil {
		exit.Error(err)
	}

	os.MkdirAll(files.ParentDir(filepath.Join(LocalWorkspace, api.Key)), os.ModePerm)
	err = files.WriteFile(apiBytes, filepath.Join(LocalWorkspace, api.Key))

	serveHostConfig := &container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: mountPath,
				Target: "/mnt/model",
			},
		},
	}

	serveContainerConfig := &container.Config{
		Image:        "cortexlabs/tf-serve:latest",
		Tty:          true,
		AttachStdout: true,
		AttachStderr: true,
		Cmd: strslice.StrSlice{
			"--port=9000", "--model_base_path=/mnt/model",
		},
		ExposedPorts: nat.PortSet{
			"9000/tcp": struct{}{},
		},
		Labels: map[string]string{
			"cortex":       "true",
			"type":         "serve",
			"apiID":        api.ID,
			"apiName":      api.Name,
			"deploymentID": api.DeploymentID,
		},
	}

	containerCreateRequest, err := DockerClient().ContainerCreate(context.Background(), serveContainerConfig, serveHostConfig, nil, "")
	if err != nil {
		return err
	}

	err = DockerClient().ContainerStart(context.Background(), containerCreateRequest.ID, dockertypes.ContainerStartOptions{})
	if err != nil {
		return err
	}

	containerInfo, err := DockerClient().ContainerInspect(context.Background(), containerCreateRequest.ID)
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
				Source: mountPath,
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
		Image:        "cortexlabs/tf-api:latest",
		Tty:          true,
		AttachStdout: true,
		AttachStderr: true,
		Env: append(
			getAPIEnv(api),
			"CORTEX_VERSION=master",
			"CORTEX_SERVING_PORT=8888",
			"CORTEX_PROVIDER=local",
			"CORTEX_CACHE_DIR="+"/mnt/cache",
			"CORTEX_MODEL_DIR="+"/mnt/model",
			"CORTEX_API_SPEC="+filepath.Join("/mnt/workspace", filepath.Base(api.Key)),
			"CORTEX_PROJECT_DIR="+"/mnt/project",
			"CORTEX_WORKERS_PER_REPLICA=1",
			"CORTEX_TF_SERVING_PORT="+"9000",
			"CORTEX_TF_SERVING_HOST="+tfContainerHost,
			"CORTEX_MAX_WORKER_CONCURRENCY=10",
			"CORTEX_SO_MAX_CONN=10",
			"CORTEX_THREADS_PER_WORKER=1",
			"AWS_ACCESS_KEY_ID="+os.Getenv("AWS_ACCESS_KEY_ID"),
			"AWS_SECRET_ACCESS_KEY="+os.Getenv("AWS_SECRET_ACCESS_KEY"),
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
	containerCreateRequest, err = DockerClient().ContainerCreate(context.Background(), apiContainerConfig, apiHostConfig, nil, "")
	if err != nil {
		return err
	}

	err = DockerClient().ContainerStart(context.Background(), containerCreateRequest.ID, dockertypes.ContainerStartOptions{})
	if err != nil {
		return err
	}

	return nil
}

func DeleteContainers(apiName string) error {
	containers, err := GetContainerByAPI(apiName)
	if err != nil {
		return err
	}

	for _, container := range containers {
		attemptErr := DockerClient().ContainerRemove(context.Background(), container.ID, dockertypes.ContainerRemoveOptions{
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

func UpdateAPI(apiConfig *userconfig.API, projectID string) (*spec.API, string, error) {
	containers, err := GetContainerByAPI(apiConfig.Name)
	if err != nil {
		return nil, "", err
	}

	deploymentID := k8s.RandomName()
	if len(containers) > 0 && containers[0].Labels["deploymentID"] != "" {
		deploymentID = containers[0].Labels["deploymentID"]
	}

	api := spec.GetAPISpec(apiConfig, projectID, deploymentID)
	if _, ok := api.Predictor.Env["PYTHONDONTWRITEBYTECODE"]; !ok {
		api.Predictor.Env["PYTHONDONTWRITEBYTECODE"] = "1"
	}

	if len(containers) == 0 {
		if err := DeployContainers(api); err != nil {
			fmt.Println(err.Error())
			DeleteContainers(api.Name)
			return nil, "", err
		}
		return api, fmt.Sprintf("creating %s", api.Name), nil
	}

	prevContainerLabels := containers[0].Labels
	if prevContainerLabels["apiName"] == api.Name && prevContainerLabels["apiID"] == api.ID {
		return api, fmt.Sprintf("%s is up to date", api.Name), nil
	}

	DeleteContainers(api.Name)
	if err := DeployContainers(api); err != nil {
		DeleteContainers(api.Name)
		return nil, "", err
	}

	return api, fmt.Sprintf("updating %s", api.Name), nil
}

func DeleteAPI(apiName string, keepCache bool) error {
	return DeleteContainers(apiName)
}
