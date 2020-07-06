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
	"strings"
	"time"

	"github.com/cortexlabs/cortex/cli/cluster"
	"github.com/cortexlabs/cortex/cli/types/cliconfig"
	"github.com/cortexlabs/cortex/pkg/lib/console"
	"github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/lib/table"
	libtime "github.com/cortexlabs/cortex/pkg/lib/time"
	"github.com/cortexlabs/cortex/pkg/lib/urls"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types"
	"github.com/cortexlabs/cortex/pkg/types/status"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
)

const (
	_titleBatchAPI = "batch api"
)

func batchAPIsTable(batchAPIs []schema.BatchAPI, envNames []string) table.Table {
	rows := make([][]interface{}, 0, len(batchAPIs))

	for i, batchAPI := range batchAPIs {
		lastUpdated := time.Unix(batchAPI.APISpec.LastUpdated, 0)
		latestStartTime := time.Time{}
		latestJobID := "-"

		for _, job := range batchAPI.JobStatuses {
			if job.StartTime.After(latestStartTime) && (job.Status == status.JobRunning || job.Status == status.JobEnqueuing) {
				latestStartTime = job.StartTime
				latestJobID = job.ID
			}
		}

		rows = append(rows, []interface{}{
			envNames[i],
			batchAPI.APISpec.Name,
			len(batchAPI.JobStatuses),
			latestJobID,
			libtime.SinceStr(&lastUpdated),
		})
	}

	return table.Table{
		Headers: []table.Header{
			{Title: _titleEnvironment},
			{Title: _titleBatchAPI},
			{Title: _titleJobCount},
			{Title: _titleLatestJobID},
			{Title: _titleLastupdated},
		},
		Rows: rows,
	}
}

func batchAPITable(batchAPI schema.BatchAPI) string {
	rows := make([][]interface{}, 0, len(batchAPI.JobStatuses))

	out := ""
	if len(batchAPI.JobStatuses) == 0 {
		out = console.Bold("no submitted jobs\n")
	} else {
		totalFailed := 0
		for _, job := range batchAPI.JobStatuses {
			succeeded := 0
			failed := 0

			if job.BatchMetrics != nil {
				failed = job.BatchMetrics.Failed
				succeeded = job.BatchMetrics.Succeeded
				totalFailed += failed
			}
			rows = append(rows, []interface{}{
				job.ID,
				job.Status.Message(),
				fmt.Sprintf("%d/%d", succeeded, job.TotalBatchCount),
				failed,
				libtime.SinceStr(&job.StartTime),
			})
		}

		t := table.Table{
			Headers: []table.Header{
				{Title: "job id"},
				{Title: "status"},
				{Title: "progress"}, // (completed/total), only show when status is running
				{Title: "failed", Hidden: totalFailed == 0},
				{Title: "start time"},
			},
			Rows: rows,
		}

		out += t.MustFormat()
	}

	apiEndpoint := urls.Join(batchAPI.BaseURL, *batchAPI.APISpec.Networking.Endpoint)
	if batchAPI.APISpec.Networking.APIGateway == userconfig.NoneAPIGatewayType {
		apiEndpoint = strings.Replace(apiEndpoint, "https://", "http://", 1)
	}

	out += "\n" + console.Bold("endpoint: ") + apiEndpoint
	out += "\n"

	out += titleStr("batch api configuration") + batchAPI.APISpec.UserStr(types.AWSProviderType)
	return out
}

func getJob(env cliconfig.Environment, apiName string, jobID string) (string, error) {
	resp, err := cluster.GetJob(MustGetOperatorConfig(env.Name), apiName, jobID)
	if err != nil {
		return "", err
	}

	out := ""

	jobStatusRow := []interface{}{
		resp.JobStatus.ID,
		resp.JobStatus.Status.Message(),
	}

	if resp.JobStatus.BatchMetrics != nil {
		avgTime := "-"
		if resp.JobStatus.BatchMetrics.AverageTimePerPartition != nil {
			avgTime = fmt.Sprintf("%.6g s", *resp.JobStatus.BatchMetrics.AverageTimePerPartition)
		}

		jobStatusRow = append(jobStatusRow,
			resp.JobStatus.TotalBatchCount,
			resp.JobStatus.BatchMetrics.Succeeded,
			resp.JobStatus.BatchMetrics.Failed,
			avgTime,
		)
	} else {
		jobStatusRow = append(jobStatusRow, "-", "-", "-", "-")
	}
	jobStatusRow = append(jobStatusRow, resp.JobStatus.StartTime.Format(time.RFC3339))

	t := table.Table{
		Headers: []table.Header{
			{Title: "job id"},
			{Title: "status"},
			{Title: "total"},
			{Title: "succeeded"},
			{Title: "failed"},
			{Title: "avg per batch"},
			{Title: "start time"},
		},
		Rows: [][]interface{}{jobStatusRow},
	}

	out = t.MustFormat() + "\n"

	if resp.JobStatus.WorkerCounts == nil {
		out += console.Bold("worker stats not available") + "\n"
	} else {
		rows := make([][]interface{}, 0, 1)
		rows = append(rows, []interface{}{
			resp.JobStatus.Parallelism,
			resp.JobStatus.WorkerCounts.Active,
			resp.JobStatus.WorkerCounts.Failed,
			resp.JobStatus.WorkerCounts.Succeeded,
		})

		t := table.Table{
			Headers: []table.Header{
				{Title: "requested"},
				{Title: "active"},
				{Title: "failed"},
				{Title: "succeeded"},
			},
			Rows: rows,
		}

		out += t.MustFormat()
	}

	jobSpecStr, err := json.Pretty(resp.JobStatus.Job)
	if err != nil {
		return "", err
	}

	out += titleStr("job configuration") + jobSpecStr

	return out, nil
}
