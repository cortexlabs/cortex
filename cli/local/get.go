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
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cortexlabs/cortex/cli/docker"
	"github.com/cortexlabs/cortex/pkg/lib/debug"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/msgpack"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types/metrics"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/status"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
)

func GetAPIs() (schema.GetAPIsResponse, error) {
	containers, err := GetAllContainers()
	if err != nil {
		return schema.GetAPIsResponse{}, err
	}

	if len(containers) == 0 {
		return schema.GetAPIsResponse{}, err
	}

	apiSet := strset.New()
	for _, container := range containers {
		apiSet.Add(container.Labels["apiName"])
	}

	apiSpecList := []spec.API{}

	for apiName := range apiSet {
		filepaths, err := files.ListDirRecursive(filepath.Join(LocalWorkspace, "apis", apiName), false)
		if err != nil {
			// TODO
		}
		var apiSpec spec.API
		found := false
		for _, filepath := range filepaths {
			if strings.HasSuffix(filepath, "spec.msgpack") {
				bytes, err := files.ReadFileBytes(filepath)
				if err != nil {
					// TODO
				}
				err = msgpack.Unmarshal(bytes, &apiSpec)
				if err != nil {
					// TODO
				}
				debug.Pp(apiSpec)
				apiSpecList = append(apiSpecList, apiSpec)
				break
			}
		}
		if !found {
			// TODO
		}
	}

	statusList := []status.Status{}
	metricsList := []metrics.Metrics{}
	for _, apiSpec := range apiSpecList {
		apiStatus, err := GetAPIStatus(&apiSpec)
		if err != nil {
			// TODO
		}

		statusList = append(statusList, apiStatus)
		metrics, err := GetAPIMetrics(&apiSpec)
		if err != nil {
			// TODO
		}
		metricsList = append(metricsList, metrics)
	}

	return schema.GetAPIsResponse{
		APIs:       apiSpecList,
		Statuses:   statusList,
		AllMetrics: metricsList,
	}, nil
}

func GetAPIMetrics(api *spec.API) (metrics.Metrics, error) {
	apiWorkspace := filepath.Join(LocalWorkspace, filepath.Dir(api.Key))

	networkStats := metrics.NetworkStats{}

	filepaths, err := files.ListDir(apiWorkspace, false)
	if err != nil {
		// TODO
	}

	totalRequestTime := 0.0
	for _, filepath := range filepaths {
		if strings.HasSuffix(filepath, "2XX") {
			fileContent, err := files.ReadFile(filepath)
			if err != nil {
				// TODO
			}

			count, err := strconv.ParseInt(fileContent, 10, 64)
			networkStats.Code2XX += int(count)
		}

		if strings.HasSuffix(filepath, "4XX") {
			fileContent, err := files.ReadFile(filepath)
			if err != nil {
				// TODO
			}

			count, err := strconv.ParseInt(fileContent, 10, 64)
			networkStats.Code4XX += int(count)
		}

		if strings.HasSuffix(filepath, "5XX") {
			fileContent, err := files.ReadFile(filepath)
			if err != nil {
				// TODO
			}

			count, err := strconv.ParseInt(fileContent, 10, 64)
			networkStats.Code5XX += int(count)
		}

		if strings.HasSuffix(filepath, "request_count") {
			fileContent, err := files.ReadFile(filepath)
			if err != nil {
				// TODO
			}

			requestTime, err := strconv.ParseFloat(fileContent, 64)
			totalRequestTime += requestTime
		}
	}

	totalRequests := networkStats.Code2XX + networkStats.Code4XX + networkStats.Code5XX
	networkStats.Total = totalRequests
	if totalRequests != 0 {
		networkStats.Latency = pointer.Float64(totalRequestTime / float64(totalRequests))
	}
	return metrics.Metrics{
		APIName:      api.Name,
		NetworkStats: &networkStats,
	}, nil
}

func GetAPIStatus(api *spec.API) (status.Status, error) {
	apiStatus := status.Status{
		APIID:   api.ID,
		APIName: api.Name,
		ReplicaCounts: status.ReplicaCounts{
			Requested: 1,
		},
	}

	containers, err := GetContainerByAPI(api.Name)
	if err != nil {
		return status.Status{}, err
	}

	if api.Predictor.Type == userconfig.TensorFlowPredictorType && len(containers) != 2 {
		apiStatus.ReplicaCounts.Updated.Failed = 1
		apiStatus.Code = status.Error
		return apiStatus, nil
	}

	if !files.IsFile(filepath.Join(LocalWorkspace, "apis", api.Name, api.ID, "api_readiness.txt")) {
		apiStatus.ReplicaCounts.Updated.Initializing = 1
		apiStatus.Code = status.Updating
		return apiStatus, nil
	}

	for _, container := range containers {
		if container.State != "running" {
			apiStatus.ReplicaCounts.Updated.Failed = 1
			apiStatus.Code = status.Error
			return apiStatus, nil
		}
	}

	apiStatus.ReplicaCounts.Updated.Ready = 1
	apiStatus.Code = status.Live
	return apiStatus, nil
}

func GetAllContainers() ([]dockertypes.Container, error) {
	dockerClient, err := docker.GetDockerClient()
	if err != nil {
		return nil, err
	}

	dargs := filters.NewArgs()
	dargs.Add("label", "cortex=true")
	containers, err := dockerClient.ContainerList(context.Background(), dockertypes.ContainerListOptions{
		All:     true,
		Filters: dargs,
	})
	if err != nil {
		return nil, err
	}

	return containers, nil
}

func FindAPISpec(apiName string) (spec.API, error) {
	apiWorkspace := filepath.Join(LocalWorkspace, "apis", apiName)
	if !files.IsDir(apiWorkspace) {
		// TODO
	}
	filepaths, err := files.ListDirRecursive(apiWorkspace, false)
	if err != nil {
		// TODO
	}

	var apiSpec spec.API
	found := false
	for _, filepath := range filepaths {
		if strings.HasSuffix(filepath, "spec.msgpack") {
			bytes, err := files.ReadFileBytes(filepath)
			if err != nil {
				// TODO
			}
			err = msgpack.Unmarshal(bytes, &apiSpec)
			if err != nil {
				// TODO
			}
			break
		}
	}
	if !found {
		// TODO
	}
	return apiSpec, nil
}

func GetAPI(apiName string) (schema.GetAPIResponse, error) {
	apiSpec, err := FindAPISpec(apiName)
	if err != nil {
		// TODO
	}

	apiStatus, err := GetAPIStatus(&apiSpec)
	if err != nil {
		// TODO
	}

	apiMetrics, err := GetAPIMetrics(&apiSpec)
	if err != nil {
		// TODO
	}

	containers, err := GetContainerByAPI(apiName)
	if err != nil {
		// TODO
	}

	if len(containers) == 0 {
		// TODO
	}
	apiContainer := containers[0]
	if len(containers) == 2 && apiContainer.Labels["type"] != "api" {
		apiContainer = containers[1]
	}

	apiPort := ""
	for _, port := range apiContainer.Ports {
		if port.PrivatePort == 8888 {
			apiPort = s.Uint16(port.PublicPort)
		}
	}

	return schema.GetAPIResponse{
		API:     apiSpec,
		Status:  apiStatus,
		Metrics: apiMetrics,
		BaseURL: "localhost:" + apiPort + "/predict",
	}, nil
}
