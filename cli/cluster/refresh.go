/*
Copyright 2021 Cortex Labs, Inc.

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
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/json"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
)

func Refresh(operatorConfig OperatorConfig, apiName string, force bool) (schema.RefreshResponse, error) {
	params := map[string]string{
		"force": s.Bool(force),
	}

	httpRes, err := HTTPPostNoBody(operatorConfig, "/refresh/"+apiName, params)
	if err != nil {
		return schema.RefreshResponse{}, err
	}

	var refreshRes schema.RefreshResponse
	err = json.Unmarshal(httpRes, &refreshRes)
	if err != nil {
		return schema.RefreshResponse{}, errors.Wrap(err, "/refresh", string(httpRes))
	}

	return refreshRes, nil
}
