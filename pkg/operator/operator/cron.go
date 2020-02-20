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
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func deleteEvictedPods() error {
	failedPods, err := config.K8s.ListPods(&kmeta.ListOptions{
		FieldSelector: "status.phase=Failed",
	})
	if err != nil {
		return err
	}

	var errs []error
	for _, pod := range failedPods {
		if pod.Status.Reason == k8s.ReasonEvicted {
			_, err := config.K8s.DeletePod(pod.Name)
			if err != nil {
				errs = append(errs, err)
			}
		}
	}

	if errors.HasError(errs) {
		return errors.FirstError(errs...)
	}
	return nil
}

func operatorTelemetry() error {
	nodes, err := config.K8s.ListNodes(nil)
	if err != nil {
		return err
	}

	instanceTypeCounts := make(map[string]int)
	var totalInstances int

	for _, node := range nodes {
		if node.Labels["workload"] != "true" {
			continue
		}

		instanceType := node.Labels["beta.kubernetes.io/instance-type"]
		if instanceType == "" {
			instanceType = "unknown"
		}

		instanceTypeCounts[instanceType]++
		totalInstances++
	}

	properties := map[string]interface{}{
		"instanceTypes": instanceTypeCounts,
		"instanceCount": totalInstances,
	}

	telemetry.Event("operator.cron", properties)

	return nil
}

func cronErrHandler(cronName string) func(error) {
	return func(err error) {
		err = errors.Wrap(err, cronName+" cron failed")
		telemetry.Error(err)
		errors.PrintError(err)
	}
}
