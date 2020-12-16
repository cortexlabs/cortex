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

package cmd

import (
	"time"

	"github.com/cortexlabs/cortex/pkg/lib/table"
	libtime "github.com/cortexlabs/cortex/pkg/lib/time"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
)

const (
	_titleTaskAPI = "task api"
	//_titleJobCount    = "running jobs"
	//_titleLatestJobID = "latest job id"
)

func taskAPIsTable(taskAPIs []schema.APIResponse, envNames []string) table.Table {
	rows := make([][]interface{}, 0, len(taskAPIs))

	for i, batchAPI := range taskAPIs {
		lastAPIUpdated := time.Unix(batchAPI.Spec.LastUpdated, 0)
		//latestStartTime := time.Time{}
		latestJobID := "-"
		runningJobs := 0

		//for _, job := range batchAPI.JobStatuses {
		//	if job.StartTime.After(latestStartTime) {
		//		latestStartTime = job.StartTime
		//		latestJobID = job.ID + fmt.Sprintf(" (submitted %s ago)", libtime.SinceStr(&latestStartTime))
		//	}
		//
		//	if job.Status.IsInProgress() {
		//		runningJobs++
		//	}
		//}

		rows = append(rows, []interface{}{
			envNames[i],
			batchAPI.Spec.Name,
			runningJobs,
			latestJobID,
			libtime.SinceStr(&lastAPIUpdated),
		})
	}

	return table.Table{
		Headers: []table.Header{
			{Title: _titleEnvironment},
			{Title: _titleTaskAPI},
			{Title: _titleJobCount},
			{Title: _titleLatestJobID},
			{Title: _titleLastupdated},
		},
		Rows: rows,
	}
}
