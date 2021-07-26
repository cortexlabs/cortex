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

package status

import (
	kapps "k8s.io/api/apps/v1"
)

type Status struct {
	Ready         int32          `json:"ready" yaml:"ready"`
	Requested     int32          `json:"requested" yaml:"requested"`
	UpToDate      int32          `json:"up_to_date" yaml:"up_to_date"`
	ReplicaCounts *ReplicaCounts `json:"replica_counts,omitempty" yaml:"replica_counts,omitempty"`
}

type ReplicaCountType string

const (
	ReplicaCountRequested      ReplicaCountType = "Requested"
	ReplicaCountPending        ReplicaCountType = "Pending"
	ReplicaCountCreating       ReplicaCountType = "Creating"
	ReplicaCountNotReady       ReplicaCountType = "NotReady"
	ReplicaCountReady          ReplicaCountType = "Ready"
	ReplicaCountReadyOutOfDate ReplicaCountType = "ReadyOutOfDate"
	ReplicaCountErrImagePull   ReplicaCountType = "ErrImagePull"
	ReplicaCountTerminating    ReplicaCountType = "Terminating"
	ReplicaCountFailed         ReplicaCountType = "Failed"
	ReplicaCountKilled         ReplicaCountType = "Killed"
	ReplicaCountKilledOOM      ReplicaCountType = "KilledOOM"
	ReplicaCountStalled        ReplicaCountType = "Stalled"
	ReplicaCountUnknown        ReplicaCountType = "Unknown"
)

var ReplicaCountTypes []ReplicaCountType = []ReplicaCountType{
	ReplicaCountRequested, ReplicaCountPending, ReplicaCountCreating,
	ReplicaCountNotReady, ReplicaCountReady, ReplicaCountReadyOutOfDate,
	ReplicaCountErrImagePull, ReplicaCountTerminating, ReplicaCountFailed,
	ReplicaCountKilled, ReplicaCountKilledOOM, ReplicaCountStalled,
	ReplicaCountUnknown,
}

type ReplicaCounts struct {
	Requested      int32 `json:"requested" yaml:"requested"`
	Pending        int32 `json:"pending" yaml:"pending"`
	Creating       int32 `json:"creating" yaml:"creating"`
	NotReady       int32 `json:"not_ready" yaml:"not_ready"`
	Ready          int32 `json:"ready" yaml:"ready"`
	ReadyOutOfDate int32 `json:"ready_out_of_date" yaml:"ready_out_of_date"`
	ErrImagePull   int32 `json:"err_image_pull" yaml:"err_image_pull"`
	Terminating    int32 `json:"terminating" yaml:"terminating"`
	Failed         int32 `json:"failed" yaml:"failed"`
	Killed         int32 `json:"killed" yaml:"killed"`
	KilledOOM      int32 `json:"killed_oom" yaml:"killed_oom"`
	Stalled        int32 `json:"stalled" yaml:"stalled"` // pending for a long time
	Unknown        int32 `json:"unknown" yaml:"unknown"`
}

// Worker counts don't have as many failure variations because Jobs clean up dead pods, so counting different failure scenarios isn't interesting
type WorkerCounts struct {
	Pending      int32 `json:"pending,omitempty" yaml:"pending,omitempty"`
	Creating     int32 `json:"creating,omitempty" yaml:"creating,omitempty"`
	NotReady     int32 `json:"not_ready,omitempty" yaml:"not_ready,omitempty"`
	Ready        int32 `json:"ready,omitempty" yaml:"ready,omitempty"`
	Succeeded    int32 `json:"succeeded,omitempty" yaml:"succeeded,omitempty"`
	ErrImagePull int32 `json:"err_image_pull,omitempty" yaml:"err_image_pull,omitempty"`
	Terminating  int32 `json:"terminating,omitempty" yaml:"terminating,omitempty"`
	Failed       int32 `json:"failed,omitempty" yaml:"failed,omitempty"`
	Killed       int32 `json:"killed,omitempty" yaml:"killed,omitempty"`
	KilledOOM    int32 `json:"killed_oom,omitempty" yaml:"killed_oom,omitempty"`
	Stalled      int32 `json:"stalled,omitempty" yaml:"stalled,omitempty"` // pending for a long time
	Unknown      int32 `json:"unknown,omitempty" yaml:"unknown,omitempty"`
}

func FromDeployment(deployment *kapps.Deployment) *Status {
	return &Status{
		Ready:     deployment.Status.ReadyReplicas,
		Requested: deployment.Status.Replicas,
		UpToDate:  deployment.Status.UpdatedReplicas,
	}
}

func (counts *ReplicaCounts) GetCountBy(replicaType ReplicaCountType) int32 {
	switch replicaType {
	case ReplicaCountRequested:
		return counts.Requested
	case ReplicaCountPending:
		return counts.Pending
	case ReplicaCountCreating:
		return counts.Creating
	case ReplicaCountNotReady:
		return counts.NotReady
	case ReplicaCountReady:
		return counts.Ready
	case ReplicaCountReadyOutOfDate:
		return counts.ReadyOutOfDate
	case ReplicaCountErrImagePull:
		return counts.ErrImagePull
	case ReplicaCountTerminating:
		return counts.Terminating
	case ReplicaCountFailed:
		return counts.Failed
	case ReplicaCountKilled:
		return counts.Killed
	case ReplicaCountKilledOOM:
		return counts.KilledOOM
	case ReplicaCountStalled:
		return counts.Stalled
	}
	return counts.Unknown
}

func (counts *ReplicaCounts) TotalFailed() int32 {
	return counts.ErrImagePull + counts.Failed + counts.Killed + counts.KilledOOM + counts.Unknown
}
