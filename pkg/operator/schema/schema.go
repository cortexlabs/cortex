/*
Copyright 2022 Cortex Labs, Inc.

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
	WorkerNodeInfos    []WorkerNodeInfo             `json:"worker_node_infos" yaml:"worker_node_infos"`
	OperatorNodeInfos  []NodeInfo                   `json:"operator_node_infos" yaml:"operator_node_infos"`
	NumPendingReplicas int                          `json:"num_pending_replicas" yaml:"num_pending_replicas"`
}

type WorkerNodeInfo struct {
	NodeInfo
	Name                 string             `json:"name" yaml:"name"`
	NumReplicas          int                `json:"num_replicas" yaml:"num_replicas"`
	NumEnqueuerReplicas  int                `json:"num_enqueuer_replicas" yaml:"num_enqueuer_replicas"`
	ComputeUserCapacity  userconfig.Compute `json:"compute_user_capacity" yaml:"compute_user_capacity"`   // the total resources available to the user on a node
	ComputeAvailable     userconfig.Compute `json:"compute_available" yaml:"compute_unavailable"`         // unused resources on a node
	ComputeUserRequested userconfig.Compute `json:"compute_user_requested" yaml:"compute_user_requested"` // total resources requested by user on a node
}

type NodeInfo struct {
	NodeGroupName string  `json:"nodegroup_name" yaml:"nodegroup_name"`
	InstanceType  string  `json:"instance_type" yaml:"instance_type"`
	IsSpot        bool    `json:"is_spot" yaml:"is_spot"`
	Price         float64 `json:"price" yaml:"price"`
}

type DeployResult struct {
	API     *APIResponse `json:"api" yaml:"api"`
	Message string       `json:"message" yaml:"message"`
	Error   string       `json:"error" yaml:"error"`
}

type APIResponse struct {
	Spec                      *spec.API               `json:"spec,omitempty" yaml:"spec,omitempty"`
	Metadata                  *spec.Metadata          `json:"metadata,omitempty"  yaml:"metadata,omitempty"`
	Status                    *status.Status          `json:"status,omitempty"  yaml:"status,omitempty"`
	NumTrafficSplitterTargets *int32                  `json:"num_traffic_splitter_targets,omitempty" yaml:"num_traffic_splitter_targets,omitempty"`
	Endpoint                  *string                 `json:"endpoint,omitempty"  yaml:"endpoint,omitempty"`
	DashboardURL              *string                 `json:"dashboard_url,omitempty"  yaml:"dashboard_url,omitempty"`
	BatchJobStatuses          []status.BatchJobStatus `json:"batch_job_statuses,omitempty"  yaml:"batch_job_statuses,omitempty"`
	TaskJobStatuses           []status.TaskJobStatus  `json:"task_job_statuses,omitempty"  yaml:"task_job_statuses,omitempty"`
	APIVersions               []APIVersion            `json:"api_versions,omitempty"  yaml:"api_versions,omitempty"`
}

type LogResponse struct {
	LogURL string `json:"log_url"`
}

type BatchJobResponse struct {
	APISpec   spec.API              `json:"api_spec" yaml:"api_spec"`
	JobStatus status.BatchJobStatus `json:"job_status" yaml:"job_status"`
	Metrics   *metrics.BatchMetrics `json:"metrics,omitempty" yaml:"metrics,omitempty"`
	Endpoint  string                `json:"endpoint" yaml:"endpoint"`
}

type TaskJobResponse struct {
	APISpec   spec.API             `json:"api_spec" yaml:"api_spec"`
	JobStatus status.TaskJobStatus `json:"job_status" yaml:"job_status"`
	Endpoint  string               `json:"endpoint" yaml:"endpoint"`
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
	APIID       string `json:"api_id" yaml:"api_id"`
	LastUpdated int64  `json:"last_updated" yaml:"last_updated"`
}

type VerifyCortexResponse struct{}

func (ir InfoResponse) GetNodesWithNodeGroupName(ngName string) []WorkerNodeInfo {
	nodesInfo := []WorkerNodeInfo{}
	for _, nodeInfo := range ir.WorkerNodeInfos {
		if nodeInfo.NodeGroupName == ngName {
			nodesInfo = append(nodesInfo, nodeInfo)
		}
	}
	return nodesInfo
}
