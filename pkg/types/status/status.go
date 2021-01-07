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

type Status struct {
	APIName       string `json:"api_name"`
	APIID         string `json:"api_id"`
	Code          Code   `json:"status_code"`
	ReplicaCounts `json:"replica_counts"`
}

type ReplicaCounts struct {
	Updated   SubReplicaCounts `json:"updated"` // fully up-to-date (compute and model)
	Stale     SubReplicaCounts `json:"stale"`
	Requested int32            `json:"requested"`
}

type SubReplicaCounts struct {
	Pending      int32 `json:"pending"`
	Initializing int32 `json:"initializing"`
	Ready        int32 `json:"ready"`
	ErrImagePull int32 `json:"err_image_pull"`
	Terminating  int32 `json:"terminating"`
	Failed       int32 `json:"failed"`
	Killed       int32 `json:"killed"`
	KilledOOM    int32 `json:"killed_oom"`
	Stalled      int32 `json:"stalled"` // pending for a long time
	Unknown      int32 `json:"unknown"`
}

// Worker counts don't have as many failure variations because Jobs clean up dead pods, so counting different failure scenarios isn't interesting
type WorkerCounts struct {
	Pending      int32 `json:"pending"`
	Initializing int32 `json:"initializing"`
	Running      int32 `json:"running"`
	Succeeded    int32 `json:"succeeded"`
	Failed       int32 `json:"failed"`
	Stalled      int32 `json:"stalled"` // pending for a long time
	Unknown      int32 `json:"unknown"`
}

func (status *Status) Message() string {
	return status.Code.Message()
}

func (src *SubReplicaCounts) TotalFailed() int32 {
	return src.Failed + src.ErrImagePull + src.Killed + src.KilledOOM + src.Stalled
}
