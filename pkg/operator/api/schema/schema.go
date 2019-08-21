/*
Copyright 2019 Cortex Labs, Inc.

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

package schema

import (
	"time"

	"github.com/cortexlabs/cortex/pkg/operator/api/context"
	"github.com/cortexlabs/cortex/pkg/operator/api/resource"
)

type DeployResponse struct {
	Message string `json:"message"`
}

type DeleteResponse struct {
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type GetResourcesResponse struct {
	Context          *context.Context                    `json:"context"`
	DataStatuses     map[string]*resource.DataStatus     `json:"data_statuses"`
	APIStatuses      map[string]*resource.APIStatus      `json:"api_statuses"`
	APIGroupStatuses map[string]*resource.APIGroupStatus `json:"api_name_statuses"`
	APIsBaseURL      string                              `json:"apis_base_url"`
}

type Deployment struct {
	Name        string                    `json:"name"`
	Status      resource.DeploymentStatus `json:"status"`
	LastUpdated time.Time                 `json:"last_updated"`
}

type GetDeploymentsResponse struct {
	Deployments []Deployment `json:"deployments"`
}

type FeatureSignature struct {
	Shape []int  `json:"shape"`
	Type  string `json:"type"`
}

type ModelInput struct {
	Signature map[string]FeatureSignature `json:"signature"`
}
