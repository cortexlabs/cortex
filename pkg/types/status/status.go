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

package status

import (
	kapps "k8s.io/api/apps/v1"
)

type Status struct {
	Ready         int32          `json:"ready" yaml:"ready"`           // deployment-reported number of ready replicas (latest + out of date)
	Requested     int32          `json:"requested" yaml:"requested"`   // deployment-reported number of requested replicas
	UpToDate      int32          `json:"up_to_date" yaml:"up_to_date"` // deployment-reported number of up-to-date replicas (in whichever phase they are found in)
	ReplicaCounts *ReplicaCounts `json:"replica_counts,omitempty" yaml:"replica_counts,omitempty"`
}

type ReplicaCountType string

const (
	ReplicaCountRequested      ReplicaCountType = "Requested"      // requested number of replicas (for up-to-date pods)
	ReplicaCountPending        ReplicaCountType = "Pending"        // pods that are in the pending state (for up-to-date pods)
	ReplicaCountCreating       ReplicaCountType = "Creating"       // pods that that have their init/non-init containers in the process of being created (for up-to-date pods)
	ReplicaCountNotReady       ReplicaCountType = "NotReady"       // pods that are not passing the readiness checks (for up-to-date pods)
	ReplicaCountReady          ReplicaCountType = "Ready"          // pods that are passing the readiness checks (for up-to-date pods)
	ReplicaCountReadyOutOfDate ReplicaCountType = "ReadyOutOfDate" // pods that are passing the readiness checks (for out-of-date pods)
	ReplicaCountErrImagePull   ReplicaCountType = "ErrImagePull"   // pods that couldn't pull the containers' images (for up-to-date pods)
	ReplicaCountTerminating    ReplicaCountType = "Terminating"    // pods that are in a terminating state (for up-to-date pods)
	ReplicaCountFailed         ReplicaCountType = "Failed"         // pods that have had their containers erroring (for up-to-date pods)
	ReplicaCountKilled         ReplicaCountType = "Killed"         // pods that have had their container processes killed (for up-to-date pods)
	ReplicaCountKilledOOM      ReplicaCountType = "KilledOOM"      // pods that have had their containers OOM (for up-to-date pods)
	ReplicaCountStalled        ReplicaCountType = "Stalled"        // pods that have been in a pending state for more than 15 mins (for up-to-date pods)
	ReplicaCountUnknown        ReplicaCountType = "Unknown"        // pods that are in an unknown state (for up-to-date pods)
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
	Terminating    int32 `json:"terminating" yaml:"terminating"` // includes up-to-date and out-of-date pods
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
