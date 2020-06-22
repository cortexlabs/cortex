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

package operator

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/cortexlabs/cortex/pkg/lib/cron"
	"github.com/cortexlabs/cortex/pkg/lib/debug"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/operator/cloud"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/operator/resources/batch"
	"github.com/cortexlabs/cortex/pkg/operator/resources/syncapi"
	"github.com/cortexlabs/cortex/pkg/types/status"
	kbatch "k8s.io/api/batch/v1"
)

func cleanupJobs() error {
	queues, err := cloud.ListQueues()
	if err != nil {
		fmt.Println(err.Error()) // TODO
	}

	jobs, err := config.K8s.ListJobs(nil)
	if err != nil {
		fmt.Println(err.Error()) // TODO
	}

	// delete if enqueue liveness failed

	jobMap := map[string]kbatch.Job{}

	for _, job := range jobs {
		jobMap[job.Labels["jobID"]] = job
	}

	for _, queueURL := range queues {
		metrics, err := cloud.GetQueueMetricsFromURL(queueURL)
		if err != nil {
			fmt.Println(err.Error()) // TODO
		}

		if metrics.InQueue+metrics.NotVisible == 0 {
			_, err := config.AWS.SQS().DeleteQueue(&sqs.DeleteQueueInput{
				QueueUrl: aws.String(queueURL),
			})
			if err != nil {
				fmt.Println(err.Error()) // TODO
			}

			// delete job
		}

		apiName, jobID := cloud.IdentifiersFromQueueURL(queueURL)

		if _, ok := jobMap[jobID]; !ok {
			// check jobspec status
			jobSpec, err := batch.DownloadJobSpec(apiName, jobID)
			if err != nil {
				fmt.Println(err.Error()) // TODO
			}

			if jobSpec.Status == status.JobEnqueuing && jobSpec.LastUpdated.Sub(time.Now()) > 2*time.Minute {
				// TODO write timeout status to job spec
				// push job spec message to cloud watch

				// delete queue
			}
		}
	}

	errs := []error{}
	for _, job := range jobs {
		// if job.Status.Active == 0 {
		queueMetrics, err := cloud.GetQueueMetrics(job.Labels["apiName"], job.Labels["jobID"])
		if err != nil {
			errs = append(errs, err)
		}

		debug.Pp(queueMetrics)
		// if queueMetrics.InQueue+queueMetrics.NotVisible == 0 {
		// 	batch.DeleteJob(job.Labels["apiName"], job.Labels["jobID"])
		// }
	}
	if len(jobs) == 0 {
		fmt.Println("empty")
	}

	return errors.FirstError(errs...)
}

func Init() error {
	telemetry.Event("operator.init")

	_, err := updateMemoryCapacityConfigMap()
	if err != nil {
		return errors.Wrap(err, "init")
	}

	deployments, err := config.K8s.ListDeploymentsWithLabelKeys("apiName")
	if err != nil {
		return err
	}

	for _, deployment := range deployments {
		// TODO only apply sync_api resource kind
		if err := syncapi.UpdateAutoscalerCron(&deployment); err != nil {
			return err
		}
	}

	cron.Run(deleteEvictedPods, cronErrHandler("delete evicted pods"), 12*time.Hour)
	cron.Run(operatorTelemetry, cronErrHandler("operator telemetry"), 1*time.Hour)
	cron.Run(cleanupJobs, cronErrHandler("operator telemetry"), 10*time.Second)

	return nil
}
