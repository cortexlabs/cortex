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
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
)

type InfoResponse struct {
	MaskedAWSAccessKeyID string                       `json:"masked_aws_access_key_id"`
	ClusterConfig        clusterconfig.InternalConfig `json:"cluster_config"`
	NodeInfos            []NodeInfo                   `json:"node_infos"`
	NumPendingReplicas   int                          `json:"num_pending_replicas"`
}

type NodeInfo struct {
	Name             string             `json:"name"`
	InstanceType     string             `json:"instance_type"`
	IsSpot           bool               `json:"is_spot"`
	Price            float64            `json:"price"`
	NumReplicas      int                `json:"num_replicas"`
	ComputeCapacity  userconfig.Compute `json:"compute_capacity"`  // the total resources available to the user on a node
	ComputeAvailable userconfig.Compute `json:"compute_available"` // unused resources on a node
}

type DeployResponse struct {
	Results []DeployResult `json:"results"`
}

type DeployResult struct {
	API     spec.API
	Message string
	Error   string
}

type GetAPIsResponse struct {
	SyncAPIs []SyncAPI `json:"sync_apis"`
}

type SyncAPI struct {
	Spec         spec.API        `json:"spec"`
	Status       status.Status   `json:"status"`
	Metrics      metrics.Metrics `json:"metrics"`
	BaseURL      string          `json:"base_url"`
	DashboardURL string          `json:"dashboard_url"`
}

type APISplitter struct {
	Spec    spec.API      `json:"spec"`
	Status  status.Status `json:"status"`
	BaseURL string        `json:"base_url"`
}

type GetAPIResponse struct {
	SyncAPI     *SyncAPI     `json:"sync_api"`
	APISplitter *APISplitter `json:"api_splitter"`
}

type DeleteResponse struct {
	Message string `json:"message"`
}

type RefreshResponse struct {
	Message string `json:"message"`
}

type ErrorResponse struct {
	Kind    string `json:"kind"`
	Message string `json:"message"`
}

type InputSignature struct {
	Shape []interface{} `json:"shape"`
	Type  string        `json:"type"`
}

type InputSignatures map[string]InputSignature

type APISummary struct {
	Message         string                     `json:"message"`
	ModelSignatures map[string]InputSignatures `json:"model_signatures"`
}
