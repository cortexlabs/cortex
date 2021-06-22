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

package schema

import (
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
	"github.com/cortexlabs/cortex/pkg/types/metrics"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/status"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
)

type InfoResponse struct {
	ClusterConfig      clusterconfig.InternalConfig `json:"cluster_config" yaml:"cluster_config"`
	NodeInfos          []NodeInfo                   `json:"node_infos" yaml:"node_infos"`
	NumPendingReplicas int                          `json:"num_pending_replicas" yaml:"num_pending_replicas"`
}

type NodeInfo struct {
	Name                    string             `json:"name" yaml:"name"`
	NodeGroupName           string             `json:"nodegroup_name" yaml:"nodegroup_name"`
	InstanceType            string             `json:"instance_type" yaml:"instance_type"`
	IsSpot                  bool               `json:"is_spot" yaml:"is_spot"`
	Price                   float64            `json:"price" yaml:"price"`
	NumReplicas             int                `json:"num_replicas" yaml:"num_replicas"`
	NumAsyncGatewayReplicas int                `json:"num_async_gateway_replicas" yaml:"num_async_gateway_replicas"`
	NumEnqueuerReplicas     int                `json:"num_enqueuer_replicas" yaml:"num_enqueuer_replicas"`
	ComputeUserCapacity     userconfig.Compute `json:"compute_user_capacity" yaml:"compute_user_capacity"`   // the total resources available to the user on a node
	ComputeAvailable        userconfig.Compute `json:"compute_available" yaml:"compute_unavailable"`         // unused resources on a node
	ComputeUserRequested    userconfig.Compute `json:"compute_user_requested" yaml:"compute_user_requested"` // total resources requested by user on a node
}

type DeployResult struct {
	API     *APIResponse `json:"api"`
	Message string       `json:"message"`
	Error   string       `json:"error"`
}

type APIResponse struct {
	Spec             spec.API                `json:"spec"`
	Status           *status.Status          `json:"status,omitempty"`
	Metrics          *metrics.Metrics        `json:"metrics,omitempty"`
	Endpoint         string                  `json:"endpoint"`
	DashboardURL     *string                 `json:"dashboard_url,omitempty"`
	BatchJobStatuses []status.BatchJobStatus `json:"batch_job_statuses,omitempty"`
	TaskJobStatuses  []status.TaskJobStatus  `json:"task_job_statuses,omitempty"`
	APIVersions      []APIVersion            `json:"api_versions,omitempty"`
}

type LogResponse struct {
	LogURL string `json:"log_url"`
}

type BatchJobResponse struct {
	APISpec   spec.API              `json:"api_spec"`
	JobStatus status.BatchJobStatus `json:"job_status"`
	Endpoint  string                `json:"endpoint"`
}

type TaskJobResponse struct {
	APISpec   spec.API             `json:"api_spec"`
	JobStatus status.TaskJobStatus `json:"job_status"`
	Endpoint  string               `json:"endpoint"`
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

type APIVersion struct {
	APIID       string `json:"api_id"`
	LastUpdated int64  `json:"last_updated"`
}

type VerifyCortexResponse struct{}

func (ir InfoResponse) GetNodesWithNodeGroupName(ngName string) []NodeInfo {
	nodesInfo := []NodeInfo{}
	for _, nodeInfo := range ir.NodeInfos {
		if nodeInfo.NodeGroupName == ngName {
			nodesInfo = append(nodesInfo, nodeInfo)
		}
	}
	return nodesInfo
}
