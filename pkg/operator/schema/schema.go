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
	Name                 string             `json:"name"`
	InstanceType         string             `json:"instance_type"`
	IsSpot               bool               `json:"is_spot"`
	Price                float64            `json:"price"`
	NumReplicas          int                `json:"num_replicas"`
	ComputeUserCapacity  userconfig.Compute `json:"compute_user_capacity"`  // the total resources available to the user on a node
	ComputeAvailable     userconfig.Compute `json:"compute_available"`      // unused resources on a node
	ComputeUserRequested userconfig.Compute `json:"compute_user_requested"` // total resources requested by user on a node
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
	RealtimeAPIs     []RealtimeAPI     `json:"realtime_apis"`
	BatchAPIs        []BatchAPI        `json:"batch_apis"`
	TrafficSplitters []TrafficSplitter `json:"traffic_splitters"`
}

type RealtimeAPI struct {
	Spec         spec.API        `json:"spec"`
	Status       status.Status   `json:"status"`
	Metrics      metrics.Metrics `json:"metrics"`
	Endpoint     string          `json:"endpoint"`
	DashboardURL string          `json:"dashboard_url"`
}

type TrafficSplitter struct {
	Spec     spec.API `json:"spec"`
	Endpoint string   `json:"endpoint"`
}

type GetAPIResponse struct {
	RealtimeAPI     *RealtimeAPI     `json:"realtime_api"`
	BatchAPI        *BatchAPI        `json:"batch_api"`
	TrafficSplitter *TrafficSplitter `json:"traffic_splitter"`
}

type BatchAPI struct {
	Spec        spec.API           `json:"spec"`
	JobStatuses []status.JobStatus `json:"job_statuses"`
	Endpoint    string             `json:"endpoint"`
}

type GetJobResponse struct {
	APISpec   spec.API         `json:"api_spec"`
	JobStatus status.JobStatus `json:"job_status"`
	Endpoint  string           `json:"endpoint"`
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
