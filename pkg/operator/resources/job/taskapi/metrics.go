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

package taskapi

import (
	"time"

	"github.com/cortexlabs/cortex/pkg/config"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/types/spec"
)

func recordSuccess(jobKey spec.JobKey) error {
	tags := []string{
		"api_name:" + jobKey.APIName,
		"job_id:" + jobKey.ID,
	}
	err := config.MetricsClient.Incr("cortex_task_succeeded", tags, 1.0)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func recordFailure(jobKey spec.JobKey) error {
	tags := []string{
		"api_name:" + jobKey.APIName,
		"job_id:" + jobKey.ID,
	}
	err := config.MetricsClient.Incr("cortex_task_failed", tags, 1.0)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func recordTimePerTask(jobKey spec.JobKey, elapsedTime time.Duration) error {
	tags := []string{
		"api_name:" + jobKey.APIName,
		"job_id:" + jobKey.ID,
	}
	err := config.MetricsClient.Histogram("cortex_time_per_task", elapsedTime.Seconds(), tags, 1.0)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}
