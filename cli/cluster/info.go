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
	"github.com/cortexlabs/cortex/pkg/operator/schema"
)

func Info(operatorConfig OperatorConfig) (*schema.InfoResponse, error) {
	httpResponse, err := HTTPGet(operatorConfig, "/info")
	if err != nil {
		return nil, errors.Wrap(err, "unable to connect to operator", "/info")
	}

	var infoResponse schema.InfoResponse
	err = json.Unmarshal(httpResponse, &infoResponse)
	if err != nil {
		return nil, errors.Wrap(err, "/info", string(httpResponse))
	}

	return &infoResponse, nil
}

func InfoGCP(operatorConfig OperatorConfig) (*schema.InfoGCPResponse, error) {
	httpResponse, err := HTTPGet(operatorConfig, "/info")
	if err != nil {
		return nil, errors.Wrap(err, "unable to connect to operator", "/info")
	}

	var infoResponse schema.InfoGCPResponse
	err = json.Unmarshal(httpResponse, &infoResponse)
	if err != nil {
		return nil, errors.Wrap(err, "/info", string(httpResponse))
	}

	return &infoResponse, nil
}
