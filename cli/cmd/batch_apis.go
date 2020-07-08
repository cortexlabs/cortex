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

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/cortexlabs/cortex/cli/cluster"
	"github.com/cortexlabs/cortex/cli/types/cliconfig"
	awslib "github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/console"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/hash"
	"github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/table"
	libtime "github.com/cortexlabs/cortex/pkg/lib/time"
	"github.com/cortexlabs/cortex/pkg/lib/urls"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types"
	"github.com/cortexlabs/cortex/pkg/types/status"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
)

const (
	_titleBatchAPI    = "batch api"
	_titleJobCount    = "running jobs"
	_titleLatestJobID = "latest job id"
)

func batchAPIsTable(batchAPIs []schema.BatchAPI, envNames []string) table.Table {
	rows := make([][]interface{}, 0, len(batchAPIs))

	for i, batchAPI := range batchAPIs {
		lastUpdated := time.Unix(batchAPI.APISpec.LastUpdated, 0)
		latestStartTime := time.Time{}
		latestJobID := "-"
		runningJobs := 0

		for _, job := range batchAPI.JobStatuses {
			if job.StartTime.After(latestStartTime) {
				latestStartTime = job.StartTime
				latestJobID = job.ID
			}

			if job.Status.IsInProgressPhase() {
				runningJobs++
			}
		}

		rows = append(rows, []interface{}{
			envNames[i],
			batchAPI.APISpec.Name,
			runningJobs,
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

			totalBatchCount := job.TotalBatchCount

			if job.Status == status.JobEnqueuing && job.QueueMetrics != nil {
				totalBatchCount = job.QueueMetrics.TotalInQueue()
			}

			if job.BatchMetrics != nil {
				failed = job.BatchMetrics.Failed
				succeeded = job.BatchMetrics.Succeeded
				totalFailed += failed
			}

			duration := ""
			if job.EndTime != nil {
				duration = job.EndTime.Sub(job.StartTime).Truncate(time.Second).String()
			} else {
				duration = time.Now().Sub(job.StartTime).Truncate(time.Second).String()
			}

			rows = append(rows, []interface{}{
				job.ID,
				job.Status.Message(),
				fmt.Sprintf("%d/%d", succeeded, totalBatchCount),
				failed,
				job.StartTime.Format(time.RFC3339),
				duration,
			})
		}

		t := table.Table{
			Headers: []table.Header{
				{Title: "job id"},
				{Title: "status"},
				{Title: "progress"}, // (completed/total), only show when status is running
				{Title: "failed", Hidden: totalFailed == 0},
				{Title: "start time"},
				{Title: "duration"},
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

	job := resp.JobStatus

	out := ""

	jobIntroTable := table.KeyValuePairs{}
	jobIntroTable.Add("job id", job.ID)
	jobIntroTable.Add("status", job.Status.Message())
	out += jobIntroTable.String(&table.KeyValuePairOpts{BoldKeys: pointer.Bool(true)})

	jobTimingTable := table.KeyValuePairs{}
	jobTimingTable.Add("start time", job.StartTime.Format(time.RFC3339))

	if job.EndTime != nil {
		jobTimingTable.Add("end time", job.EndTime.Format(time.RFC3339))
		jobTimingTable.Add("duration", job.EndTime.Sub(job.StartTime).Truncate(time.Second).String())
	} else {
		jobTimingTable.Add("end time", "-")
		jobTimingTable.Add("duration", time.Now().Sub(job.StartTime).Truncate(time.Second).String())
	}

	out += "\n" + jobTimingTable.String(&table.KeyValuePairOpts{BoldKeys: pointer.Bool(true)})

	totalBatchCount := job.TotalBatchCount

	if job.Status == status.JobEnqueuing && job.QueueMetrics != nil {
		totalBatchCount = job.QueueMetrics.TotalInQueue()
	}

	succeeded := "-"
	failed := "-"
	avgTimePerBatch := "-"

	if job.BatchMetrics != nil {
		if job.BatchMetrics.AverageTimePerBatch != nil {
			avgTimePerBatch = fmt.Sprintf("%.6g s", *job.BatchMetrics.AverageTimePerBatch)
		}

		succeeded = s.Int(job.BatchMetrics.Succeeded)
		failed = s.Int(job.BatchMetrics.Failed)
	}

	t := table.Table{
		Headers: []table.Header{
			{Title: "total"},
			{Title: "succeeded"},
			{Title: "failed"},
			{Title: "avg time per batch"},
		},
		Rows: [][]interface{}{
			{
				totalBatchCount,
				succeeded,
				failed,
				avgTimePerBatch,
			},
		},
	}

	out += titleStr("batch stats") + t.MustFormat(&table.Opts{BoldHeader: pointer.Bool(false)})

	out += titleStr("worker stats")
	if job.WorkerCounts == nil {
		out += "\nworker stats not available"
	} else {
		workers, _ := job.RequestedWorkers()
		rows := make([][]interface{}, 0, 1)
		rows = append(rows, []interface{}{
			workers,
			job.WorkerCounts.Active,
			job.WorkerCounts.Failed,
			job.WorkerCounts.Succeeded,
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

		out += t.MustFormat(&table.Opts{BoldHeader: pointer.Bool(false)})
	}

	jobSpecStr, err := json.Pretty(job.Job)
	if err != nil {
		return "", err
	}

	out += titleStr("job configuration") + jobSpecStr

	return out, nil
}

func DeleteQueues(clusterName string, awsClient *awslib.Client) error {
	queuePrefix := hash.String(clusterName)[:11] + "-"

	output, err := awsClient.SQS().ListQueues(
		&sqs.ListQueuesInput{
			QueueNamePrefix: aws.String(queuePrefix),
		},
	)
	if err != nil {
		return errors.WithStack(err)
	}

	for _, queueURL := range output.QueueUrls {
		awsClient.SQS().DeleteQueue(&sqs.DeleteQueueInput{ // best effort delete
			QueueUrl: queueURL,
		})
	}

	return nil
}
