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
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/docker"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types/spec"
)

func GetAPIs() (schema.GetAPIsResponse, error) {
	_, err := docker.GetDockerClient()
	if err != nil {
		return schema.GetAPIsResponse{}, err
	}

	apiSpecList, err := ListAPISpecs()
	if err != nil {
		return schema.GetAPIsResponse{}, err
	}

	realtimeAPIs := make([]schema.RealtimeAPI, len(apiSpecList))
	for i, apiSpec := range apiSpecList {
		apiStatus, err := GetAPIStatus(&apiSpec)
		if err != nil {
			return schema.GetAPIsResponse{}, err
		}

		metrics, err := GetAPIMetrics(&apiSpec)
		if err != nil {
			return schema.GetAPIsResponse{}, err
		}

		realtimeAPIs[i] = schema.RealtimeAPI{
			Spec:    apiSpec,
			Status:  apiStatus,
			Metrics: metrics,
		}
	}

	return schema.GetAPIsResponse{
		RealtimeAPIs: realtimeAPIs,
	}, nil
}

func ListAPISpecs() ([]spec.API, error) {
	filepaths, err := files.ListDirRecursive(filepath.Join(_localWorkspaceDir, "apis"), false)
	if err != nil {
		return nil, err
	}

	apiSpecList := []spec.API{}
	for _, specPath := range filepaths {
		if !strings.HasSuffix(filepath.Base(specPath), "-spec.json") {
			continue
		}

		apiSpecVersion := GetVersionFromAPISpecFilePath(specPath)
		if apiSpecVersion != consts.CortexVersion {
			continue
		}

		var apiSpec spec.API
		bytes, err := files.ReadFileBytes(specPath)
		if err != nil {
			return nil, errors.Wrap(err, "api", specPath)
		}
		err = json.Unmarshal(bytes, &apiSpec)
		if err != nil {
			return nil, errors.Wrap(err, "api", specPath)
		}
		apiSpecList = append(apiSpecList, apiSpec)
	}

	return apiSpecList, nil
}

func ListVersionMismatchedAPIs() ([]string, error) {
	filepaths, err := files.ListDirRecursive(filepath.Join(_localWorkspaceDir, "apis"), false)
	if err != nil {
		return nil, err
	}

	apiNames := []string{}
	for _, specPath := range filepaths {
		// Check msgpack for compatibility
		if !strings.HasSuffix(filepath.Base(specPath), "-spec.json") && !strings.HasSuffix(filepath.Base(specPath), "-spec.msgpack") {
			continue
		}
		apiSpecVersion := GetVersionFromAPISpecFilePath(specPath)
		if apiSpecVersion == consts.CortexVersion {
			continue
		}

		key, err := filepath.Rel(filepath.Join(_localWorkspaceDir, "apis"), specPath)
		if err != nil {
			return nil, err
		}
		splitKey := strings.Split(key, "/")
		if len(splitKey) == 0 {
			continue
		}
		apiNames = append(apiNames, splitKey[0])
	}
	return apiNames, nil
}

func GetAPI(apiName string) (schema.GetAPIResponse, error) {
	_, err := docker.GetDockerClient()
	if err != nil {
		return schema.GetAPIResponse{}, err
	}

	apiSpec, err := FindAPISpec(apiName)
	if err != nil {
		return schema.GetAPIResponse{}, err
	}

	apiStatus, err := GetAPIStatus(apiSpec)
	if err != nil {
		return schema.GetAPIResponse{}, err
	}

	apiMetrics, err := GetAPIMetrics(apiSpec)
	if err != nil {
		return schema.GetAPIResponse{}, err
	}

	containers, err := GetContainersByAPI(apiName)
	if err != nil {
		return schema.GetAPIResponse{}, err
	}

	if len(containers) == 0 {
		return schema.GetAPIResponse{}, ErrorAPIContainersNotFound(apiName)
	}
	apiContainer := containers[0]
	if len(containers) == 2 && apiContainer.Labels["type"] != "api" {
		apiContainer = containers[1]
	}

	apiPort := apiSpec.Networking.LocalPort

	return schema.GetAPIResponse{
		RealtimeAPI: &schema.RealtimeAPI{
			Spec:     *apiSpec,
			Status:   apiStatus,
			Metrics:  apiMetrics,
			Endpoint: fmt.Sprintf("http://localhost:%d", *apiPort),
		},
	}, nil
}
