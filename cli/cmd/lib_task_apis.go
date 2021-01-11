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
	_titleTaskJobCount    = "running jobs"
	_titleLatestTaskJobID = "latest job id"
)

func taskAPIsTable(taskAPIs []schema.APIResponse, envNames []string) table.Table {
	rows := make([][]interface{}, 0, len(taskAPIs))

	for i, taskAPI := range taskAPIs {
		lastAPIUpdated := time.Unix(taskAPI.Spec.LastUpdated, 0)
		latestStartTime := time.Time{}
		latestJobID := "-"
		runningJobs := 0

		for _, job := range taskAPI.TaskJobStatuses {
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
	jobRows := make([][]interface{}, 0, len(taskAPI.TaskJobStatuses))

	out := ""
	if len(taskAPI.TaskJobStatuses) == 0 {
		out = console.Bold("no submitted task jobs\n")
	} else {
		for _, job := range taskAPI.TaskJobStatuses {
			jobEndTime := time.Now()
			if job.EndTime != nil {
				jobEndTime = *job.EndTime
			}

			duration := jobEndTime.Sub(job.StartTime).Truncate(time.Second).String()

			jobRows = append(jobRows, []interface{}{
				job.ID,
				job.Status.Message(),
				job.StartTime.Format(_timeFormat),
				duration,
			})
		}

		t := table.Table{
			Headers: []table.Header{
				{Title: "task job id"},
				{Title: "status"},
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
