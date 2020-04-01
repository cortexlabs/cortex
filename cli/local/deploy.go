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

	"github.com/cortexlabs/cortex/pkg/lib/debug"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/msgpack"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"github.com/docker/docker/api/types"
	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"
)

func GetContainerByAPI(apiName string) ([]dockertypes.Container, error) {
	dargs := filters.NewArgs()
	dargs.Add("label", "cortex=true")
	dargs.Add("label", "apiName="+apiName)

	containers, err := DockerClient().ContainerList(context.Background(), types.ContainerListOptions{
		Filters: dargs,
	})
	if err != nil {
		return nil, err
	}

	return containers, nil
}

// func deploymentSpec(api *spec.API, prevDeployment *kapps.Deployment) *kapps.Deployment {
// 	switch api.Predictor.Type {
// 	case userconfig.TensorFlowPredictorType:
// 		return tfAPISpec(api, prevDeployment)
// 	case userconfig.ONNXPredictorType:
// 		return onnxAPISpec(api, prevDeployment)
// 	case userconfig.PythonPredictorType:
// 		return pythonAPISpec(api, prevDeployment)
// 	default:
// 		return nil // unexpected
// 	}
// }

func DeployContainers(api *spec.API) error {
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
				Source: filepath.Join(LocalWorkspace),
				Target: "/mnt/workspace",
			},
		},
	}

	containerConfig := &container.Config{
		Image:        "cortexlabs/python-serve:latest",
		Tty:          true,
		AttachStdout: true,
		AttachStderr: true,
		Env: []string{
			"CORTEX_VERSION=master",
			"CORTEX_SERVING_PORT=8888",
			"CORTEX_BASE_DIR=" + "/mnt/workspace",
			"CORTEX_CACHE_DIR=" + "/mnt/cache",
			"CORTEX_API_SPEC=" + filepath.Join("/mnt/workspace", api.Key),
			"CORTEX_PROJECT_DIR=" + "/mnt/project",
			"CORTEX_WORKERS_PER_REPLICA=1",
			"CORTEX_MAX_WORKER_CONCURRENCY=10",
			"CORTEX_SO_MAX_CONN=10",
			"CORTEX_THREADS_PER_WORKER=1",
			"AWS_ACCESS_KEY_ID=" + os.Getenv("AWS_ACCESS_KEY_ID"),
			"AWS_SECRET_ACCESS_KEY=" + os.Getenv("AWS_SECRET_ACCESS_KEY"),
		},
		ExposedPorts: nat.PortSet{
			"8888/tcp": struct{}{},
		},
		Labels: map[string]string{
			"cortex":       "true",
			"apiID":        api.ID,
			"apiName":      api.Name,
			"deploymentID": api.DeploymentID,
		},
	}
	debug.Pp(containerConfig.Labels)
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

// func ONNXSpec(api *spec.API) error {
// 	modelPath := *api.Predictor.Model

// 	if strings.HasPrefix()

// 	hostConfig := &container.HostConfig{
// 		PortBindings: nat.PortMap{
// 			"8888/tcp": []nat.PortBinding{
// 				{
// 					// HostPort: "8888",
// 				},
// 			},
// 		},
// 		Mounts: []mount.Mount{
// 			{
// 				Type:   mount.TypeBind,
// 				Source: CWD,
// 				Target: "/mnt/project",
// 			},
// 			{
// 				Type:   mount.TypeBind,
// 				Source: filepath.Join(LocalWorkspace),
// 				Target: "/mnt/workspace",
// 			},
// 		},
// 	}

// 	containerConfig := &container.Config{
// 		Image:        "cortexlabs/onnx-serve:latest",
// 		Tty:          true,
// 		AttachStdout: true,
// 		AttachStderr: true,
// 		Env: []string{
// 			"CORTEX_VERSION=master",
// 			"CORTEX_SERVING_PORT=8888",
// 			"CORTEX_BASE_DIR=" + "/mnt/workspace",
// 			"CORTEX_CACHE_DIR=" + "/mnt/cache",
// 			"CORTEX_API_SPEC=" + filepath.Join("/mnt/workspace", api.Key),
// 			"CORTEX_PROJECT_DIR=" + "/mnt/project",
// 			"CORTEX_WORKERS_PER_REPLICA=1",
// 			"CORTEX_MAX_WORKER_CONCURRENCY=10",
// 			"CORTEX_SO_MAX_CONN=10",
// 			"CORTEX_THREADS_PER_WORKER=1",
// 			"AWS_ACCESS_KEY_ID=" + os.Getenv("AWS_ACCESS_KEY_ID"),
// 			"AWS_SECRET_ACCESS_KEY=" + os.Getenv("AWS_SECRET_ACCESS_KEY"),
// 		},
// 		ExposedPorts: nat.PortSet{
// 			"8888/tcp": struct{}{},
// 		},
// 		Labels: map[string]string{
// 			"cortex":       "true",
// 			"apiID":        api.ID,
// 			"apiName":      api.Name,
// 			"deploymentID": api.DeploymentID,
// 		},
// 	}
// 	debug.Pp(containerConfig.Labels)
// 	containerInfo, err := DockerClient().ContainerCreate(context.Background(), containerConfig, hostConfig, nil, "")
// 	if err != nil {
// 		return err
// 	}

// 	err = DockerClient().ContainerStart(context.Background(), containerInfo.ID, dockertypes.ContainerStartOptions{})
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }

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

	if len(containers) == 0 {
		apiBytes, err := msgpack.Marshal(api)
		if err != nil {
			exit.Error(err)
		}
		os.MkdirAll(files.ParentDir(filepath.Join(LocalWorkspace, api.Key)), os.ModePerm)
		err = files.WriteFile(apiBytes, filepath.Join(LocalWorkspace, api.Key))

		if err := DeployContainers(api); err != nil {
			DeleteContainers(api.Name)
			return nil, "", err
		}
		return api, fmt.Sprintf("creating %s", api.Name), nil
	}

	prevContainerLabels := containers[0].Labels
	debug.Pp(prevContainerLabels)
	if prevContainerLabels["apiName"] == api.Name && prevContainerLabels["apiID"] == api.ID {
		return api, fmt.Sprintf("%s is up to date", api.Name), nil
	}

	apiBytes, err := msgpack.Marshal(api)
	if err != nil {
		exit.Error(err)
	}
	os.MkdirAll(files.ParentDir(filepath.Join(LocalWorkspace, api.Key)), os.ModePerm)
	err = files.WriteFile(apiBytes, filepath.Join(LocalWorkspace, api.Key))

	DeleteContainers(api.Name)
	if err := DeployContainers(api); err != nil {
		panic("here")
		fmt.Println("here")
		DeleteContainers(api.Name)
		return nil, "", err
	}

	return api, fmt.Sprintf("updating %s", api.Name), nil
}

func DeleteAPI(apiName string, keepCache bool) error {
	return DeleteContainers(apiName)
}
