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
	"fmt"
	"path"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/lib/prompt"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
)

func Delete(operatorConfig OperatorConfig, apiName string, keepCache bool, force bool) (schema.DeleteResponse, error) {
	if !force {
		readyReplicas := getReadyRealtimeAPIReplicasOrNil(operatorConfig, apiName)
		if readyReplicas != nil && *readyReplicas > 2 {
			prompt.YesOrExit(fmt.Sprintf("are you sure you want to delete %s (which has %d live replicas)?", apiName, *readyReplicas), "", "")
		}
	}

	params := map[string]string{
		"apiName":   apiName,
		"keepCache": s.Bool(keepCache),
	}

	httpRes, err := HTTPDelete(operatorConfig, "/delete/"+apiName, params)
	if err != nil {
		return schema.DeleteResponse{}, err
	}

	var deleteRes schema.DeleteResponse
	err = json.Unmarshal(httpRes, &deleteRes)
	if err != nil {
		return schema.DeleteResponse{}, errors.Wrap(err, "/delete", string(httpRes))
	}

	return deleteRes, nil
}

func getReadyRealtimeAPIReplicasOrNil(operatorConfig OperatorConfig, apiName string) *int32 {
	httpRes, err := HTTPGet(operatorConfig, "/get/"+apiName)
	if err != nil {
		return nil
	}

	var apiRes schema.GetAPIResponse
	if err = json.Unmarshal(httpRes, &apiRes); err != nil {
		return nil
	}

	if apiRes.RealtimeAPI == nil {
		return nil
	}

	totalReady := apiRes.RealtimeAPI.Status.Updated.Ready + apiRes.RealtimeAPI.Status.Stale.Ready
	return &totalReady
}

func StopJob(operatorConfig OperatorConfig, apiName string, jobID string) (schema.DeleteResponse, error) {
	params := map[string]string{
		"apiName": apiName,
		"jobID":   jobID,
	}

	httpRes, err := HTTPDelete(operatorConfig, path.Join("/batch", apiName, jobID), params)
	if err != nil {
		return schema.DeleteResponse{}, err
	}

	var deleteRes schema.DeleteResponse
	err = json.Unmarshal(httpRes, &deleteRes)
	if err != nil {
		return schema.DeleteResponse{}, errors.Wrap(err, string(httpRes))
	}

	return deleteRes, nil
}
