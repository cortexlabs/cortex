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

package schema

import (
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
	"github.com/cortexlabs/cortex/pkg/types/metrics"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/status"
)

type InfoResponse struct {
	MaskedAWSAccessKeyID string                        `json:"masked_aws_access_key_id"`
	ClusterConfig        *clusterconfig.InternalConfig `json:"cluster_config"`
}

type DeployResponse struct {
	Results []DeployResult `json:"results"`
	BaseURL string         `json:"base_url"`
	Message string         `json:"message"`
}

type DeployResult struct {
	API     *spec.API
	Message string
	Error   string
}

type GetAPIsResponse struct {
	APIs       []spec.API        `json:"apis"`
	Statuses   []status.Status   `json:"statuses"`
	AllMetrics []metrics.Metrics `json:"all_metrics"`
	BaseURL    string            `json:"base_url"`
}

type GetAPIResponse struct {
	API     *spec.API        `json:"api"`
	Status  *status.Status   `json:"status"`
	Metrics *metrics.Metrics `json:"metrics"`
	BaseURL string           `json:"base_url"`
}

type DeleteResponse struct {
	Message string `json:"message"`
}

type RefreshResponse struct {
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type FeatureSignature struct {
	Shape []interface{} `json:"shape"`
	Type  string        `json:"type"`
}

type APISummary struct {
	Message        string                      `json:"message"`
	ModelSignature map[string]FeatureSignature `json:"model_signature"`
}
