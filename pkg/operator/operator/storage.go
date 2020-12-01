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

package operator

import (
	"github.com/cortexlabs/cortex/pkg/lib/cast"
	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/debug"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/parallel"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/types/spec"
)

func DownloadAPISpec(apiName string, apiID string) (*spec.API, error) {
	s3Key := spec.Key(apiName, apiID, config.Cluster.ClusterName)
	var api spec.API

	if err := config.AWS.ReadJSONFromS3(&api, config.Cluster.Bucket, s3Key); err != nil {
		return nil, err
	}

	return &api, nil
}

func DownloadRawAPISpec(api *spec.API) (map[string]interface{}, error) {
	s3Key := api.RawAPIKey(config.Cluster.ClusterName)

	configBytes, err := config.AWS.ReadBytesFromS3(config.Cluster.Bucket, s3Key)
	if err != nil {
		return nil, err
	}

	configData, err := cr.ReadYAMLBytes(configBytes)
	if err != nil {
		return nil, err
	}

	debug.Ppj(configData)

	apiSlice, ok := cast.InterfaceToInterfaceSlice(configData)
	if !ok {
		return nil, errors.ErrorUnexpected("unable to case userconfig to json slice map")
	}

	if len(apiSlice) == 0 {
		return nil, errors.ErrorUnexpected("unable to case userconfig to json slice map")
	}

	strMap, ok := cast.InterfaceToStrInterfaceMapRecursive(apiSlice[0])
	if !ok {
		return nil, errors.ErrorUnexpected("unable to case userconfig to json slice map")
	}

	return strMap, nil
}

func DownloadAPISpecs(apiNames []string, apiIDs []string) ([]spec.API, error) {
	apis := make([]spec.API, len(apiNames))
	fns := make([]func() error, len(apiNames))

	for i := range apiNames {
		localIdx := i
		fns[i] = func() error {
			api, err := DownloadAPISpec(apiNames[localIdx], apiIDs[localIdx])
			if err != nil {
				return err
			}
			apis[localIdx] = *api
			return nil
		}
	}

	if len(fns) > 0 {
		err := parallel.RunFirstErr(fns[0], fns[1:]...)
		if err != nil {
			return nil, err
		}
	}

	return apis, nil
}
