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

package cluster

import (
	"path"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
)

func GetAPIs(operatorConfig OperatorConfig) (schema.GetAPIsResponse, error) {
	httpRes, err := HTTPGet(operatorConfig, "/get")
	if err != nil {
		return schema.GetAPIsResponse{}, err
	}

	var apisRes schema.GetAPIsResponse
	if err = json.Unmarshal(httpRes, &apisRes); err != nil {
		return schema.GetAPIsResponse{}, errors.Wrap(err, "/get", string(httpRes))
	}
	return apisRes, nil
}

func GetAPI(operatorConfig OperatorConfig, apiName string) (schema.GetAPIResponse, error) {
	httpRes, err := HTTPGet(operatorConfig, "/get/"+apiName)
	if err != nil {
		return schema.GetAPIResponse{}, err
	}

	var apiRes schema.GetAPIResponse
	if err = json.Unmarshal(httpRes, &apiRes); err != nil {
		return schema.GetAPIResponse{}, errors.Wrap(err, "/get/"+apiName, string(httpRes))
	}

	return apiRes, nil
}

func GetJob(operatorConfig OperatorConfig, apiName string, jobID string) (schema.GetJobResponse, error) {
	endpoint := path.Join("/batch", apiName, jobID)
	httpRes, err := HTTPGet(operatorConfig, endpoint)
	if err != nil {
		return schema.GetJobResponse{}, err
	}

	var jobRes schema.GetJobResponse
	if err = json.Unmarshal(httpRes, &jobRes); err != nil {
		return schema.GetJobResponse{}, errors.Wrap(err, endpoint, string(httpRes))
	}

	return jobRes, nil
}
