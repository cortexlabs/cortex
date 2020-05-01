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
	"strings"
	"time"

	"github.com/cortexlabs/cortex/pkg/lib/docker"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/types/metrics"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/status"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
)

func GetAPIMetrics(api *spec.API) (metrics.Metrics, error) {
	apiWorkspace := filepath.Join(_localWorkspaceDir, filepath.Dir(api.Key))

	networkStats := metrics.NetworkStats{}

	filepaths, err := files.ListDir(apiWorkspace, false)
	if err != nil {
		return metrics.Metrics{}, errors.Wrap(err, "api", api.Name)
	}

	totalRequestTime := 0.0
	for _, filepath := range filepaths {
		if strings.HasSuffix(filepath, ".2XX") {
			fileContent, err := files.ReadFile(filepath)
			if err != nil {
				return metrics.Metrics{}, errors.Wrap(err, "api", api.Name)
			}
			count, ok := s.ParseInt(fileContent)
			if !ok {
				count = 0
			}
			networkStats.Code2XX += int(count)
		}

		if strings.HasSuffix(filepath, ".4XX") {
			fileContent, err := files.ReadFile(filepath)
			if err != nil {
				return metrics.Metrics{}, errors.Wrap(err, "api", api.Name)
			}

			count, ok := s.ParseInt(fileContent)
			if !ok {
				count = 0
			}
			networkStats.Code4XX += int(count)
		}

		if strings.HasSuffix(filepath, ".5XX") {
			fileContent, err := files.ReadFile(filepath)
			if err != nil {
				return metrics.Metrics{}, errors.Wrap(err, "api", api.Name)
			}

			count, ok := s.ParseInt(fileContent)
			if !ok {
				count = 0
			}
			networkStats.Code5XX += int(count)
		}

		if strings.HasSuffix(filepath, ".request_time") {
			fileContent, err := files.ReadFile(filepath)
			if err != nil {
				return metrics.Metrics{}, errors.Wrap(err, "api", api.Name)
			}

			requestTime, ok := s.ParseFloat64(fileContent)
			if !ok {
				requestTime = 0
			}
			totalRequestTime += requestTime
		}
	}

	totalRequests := networkStats.Code2XX + networkStats.Code4XX + networkStats.Code5XX
	networkStats.Total = totalRequests
	if totalRequests != 0 && totalRequestTime != 0 {
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

	// 10 second grace period between creating api spec file and looking for containers
	if api.LastUpdated+10 > time.Now().Unix() {
		apiStatus.ReplicaCounts.Updated.Initializing = 1
		apiStatus.Code = status.Updating
		return apiStatus, nil
	}

	containers, err := GetContainersByAPI(api.Name)
	if err != nil {
		return status.Status{}, err
	}

	if len(containers) == 0 {
		apiStatus.ReplicaCounts.Updated.Failed = 1
		apiStatus.Code = status.Error
		return apiStatus, nil
	}

	if api.Predictor.Type == userconfig.TensorFlowPredictorType && len(containers) != 2 {
		apiStatus.ReplicaCounts.Updated.Failed = 1
		apiStatus.Code = status.Error
		return apiStatus, nil
	}

	for _, container := range containers {
		if container.State != "running" {
			dockerClient := docker.MustDockerClient()
			containerInfo, err := dockerClient.ContainerInspect(context.Background(), container.ID)
			if err != nil {
				return status.Status{}, errors.Wrap(err, api.Identify())
			}
			if containerInfo.State.OOMKilled || containerInfo.State.ExitCode == 137 {
				apiStatus.ReplicaCounts.Updated.Failed = 1
				apiStatus.Code = status.OOM
				return apiStatus, nil
			}

			apiStatus.ReplicaCounts.Updated.Failed = 1
			apiStatus.Code = status.Error
			return apiStatus, nil
		}
	}

	if !files.IsFile(filepath.Join(_localWorkspaceDir, filepath.Dir(api.Key), "api_readiness.txt")) {
		apiStatus.ReplicaCounts.Updated.Initializing = 1
		apiStatus.Code = status.Updating
		return apiStatus, nil
	}

	apiStatus.ReplicaCounts.Updated.Ready = 1
	apiStatus.Code = status.Live
	return apiStatus, nil
}
