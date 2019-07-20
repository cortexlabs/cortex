/*
Copyright 2019 Cortex Labs, Inc.

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

package workloads

import (
	"time"

	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/operator/config"
)

const cronInterval = 5 // seconds

var cronChannel = make(chan struct{}, 1)

func cronRunner() {
	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-cronChannel:
			runCron()
		case <-timer.C:
			runCron()
		}
		timer.Reset(5 * time.Second)
	}
}

func runCronNow() {
	cronChannel <- struct{}{}
}

func runCron() {
	defer reportAndRecover("cron failed")

	if err := UpdateWorkflows(); err != nil {
		config.Telemetry.ReportError(err)
		errors.PrintError(err)
	}

	apiPods, err := config.Kubernetes.ListPodsByLabels(map[string]string{
		"workloadType": workloadTypeAPI,
		"userFacing":   "true",
	})
	if err != nil {
		config.Telemetry.ReportError(err)
		errors.PrintError(err)
	}

	if err := updateAPISavedStatuses(apiPods); err != nil {
		config.Telemetry.ReportError(err)
		errors.PrintError(err)
	}

	if err := uploadLogPrefixesFromAPIPods(apiPods); err != nil {
		config.Telemetry.ReportError(err)
		errors.PrintError(err)
	}

	failedPods, err := config.Kubernetes.ListPods(&kmeta.ListOptions{
		FieldSelector: "status.phase=Failed",
	})
	if err != nil {
		config.Telemetry.ReportError(err)
		errors.PrintError(err)
	}

	if err := updateDataWorkloadErrors(failedPods); err != nil {
		config.Telemetry.ReportError(err)
		errors.PrintError(err)
	}
}

func reportAndRecover(strs ...string) error {
	if errInterface := recover(); errInterface != nil {
		err := errors.CastRecoverError(errInterface, strs...)
		config.Telemetry.ReportError(err)
		errors.PrintError(err)
		return err
	}
	return nil
}
