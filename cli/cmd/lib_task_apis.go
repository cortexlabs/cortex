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
	"fmt"
	"time"

	"github.com/cortexlabs/cortex/pkg/lib/console"
	"github.com/cortexlabs/cortex/pkg/lib/table"
	libtime "github.com/cortexlabs/cortex/pkg/lib/time"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types"
)

const (
	_titleTaskAPI         = "task api"
	_titleTaskJobCount    = "running task jobs"
	_titleLatestTaskJobID = "latest task job id"
)

func taskAPIsTable(taskAPIs []schema.APIResponse, envNames []string) table.Table {
	rows := make([][]interface{}, 0, len(taskAPIs))

	for i, taskAPI := range taskAPIs {
		// FIXME: should iterate over task jobs instead of batch jobs
		lastAPIUpdated := time.Unix(taskAPI.Spec.LastUpdated, 0)
		latestStartTime := time.Time{}
		latestJobID := "-"
		runningJobs := 0

		for _, job := range taskAPI.JobStatuses {
			if job.StartTime.After(latestStartTime) {
				latestStartTime = job.StartTime
				latestJobID = job.ID + fmt.Sprintf(" (submitted %s ago)", libtime.SinceStr(&latestStartTime))
			}

			if job.Status.IsInProgress() {
				runningJobs++
			}
		}

		rows = append(rows, []interface{}{
			envNames[i],
			taskAPI.Spec.Name,
			runningJobs,
			latestJobID,
			libtime.SinceStr(&lastAPIUpdated),
		})
	}

	return table.Table{
		Headers: []table.Header{
			{Title: _titleEnvironment},
			{Title: _titleTaskAPI},
			{Title: _titleTaskJobCount},
			{Title: _titleLatestTaskJobID},
			{Title: _titleLastupdated},
		},
		Rows: rows,
	}
}

func taskAPITable(taskAPI schema.APIResponse) string {
	jobRows := make([][]interface{}, 0, len(taskAPI.JobStatuses))

	out := ""
	if len(taskAPI.JobStatuses) == 0 {
		out = console.Bold("no submitted task jobs\n")
	} else {
		// FIXME: should access task metrics instead of batch metrics
		totalFailed := 0
		for _, job := range taskAPI.JobStatuses {
			failed := 0

			if job.BatchMetrics != nil {
				failed = job.BatchMetrics.Failed
				totalFailed += failed
			}

			jobEndTime := time.Now()
			if job.EndTime != nil {
				jobEndTime = *job.EndTime
			}

			duration := jobEndTime.Sub(job.StartTime).Truncate(time.Second).String()

			jobRows = append(jobRows, []interface{}{
				job.ID,
				job.Status.Message(),
				failed,
				job.StartTime.Format(_timeFormat),
				duration,
			})
		}

		t := table.Table{
			Headers: []table.Header{
				{Title: "task job id"},
				{Title: "status"},
				{Title: "failed", Hidden: totalFailed == 0},
				{Title: "start time"},
				{Title: "duration"},
			},
			Rows: jobRows,
		}

		out += t.MustFormat()
	}

	out += "\n" + console.Bold("endpoint: ") + taskAPI.Endpoint + "\n"

	out += "\n" + apiHistoryTable(taskAPI.APIVersions)

	if !_flagVerbose {
		return out
	}

	out += titleStr("task api configuration") + taskAPI.Spec.UserStr(types.AWSProviderType)

	return out
}
