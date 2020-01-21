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
	"time"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/parallel"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	kapps "k8s.io/api/apps/v1"
	kautoscaling "k8s.io/api/autoscaling/v2beta2"
	kcore "k8s.io/api/core/v1"
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

func updateHPAs() error {
	var deployments []kapps.Deployment
	var hpas []kautoscaling.HorizontalPodAutoscaler
	err := parallel.RunFirstErr(
		func() error {
			var err error
			deployments, err = config.K8s.ListDeploymentsWithLabelKeys("apiName")
			return err
		},
		func() error {
			var err error
			hpas, err = config.K8s.ListHPAsWithLabelKeys("apiName")
			return err
		},
	)
	if err != nil {
		return err
	}

	hpaMap := make(map[string]*kautoscaling.HorizontalPodAutoscaler, len(hpas))
	for _, hpa := range hpas {
		hpaMap[hpa.Name] = &hpa
	}

	var allPods []kcore.Pod
	var errs []error

	for _, deployment := range deployments {
		if hpaMap[deployment.Name] != nil {
			continue // since the HPA is deleted every time the deployment is updated
		}

		if allPods == nil {
			var err error
			allPods, err = config.K8s.ListPodsWithLabelKeys("apiName")
			if err != nil {
				return err
			}
		}

		replicaCounts := getReplicaCounts(&deployment, allPods)
		if deployment.Spec.Replicas == nil || replicaCounts.Updated.Ready < *deployment.Spec.Replicas {
			continue // not yet up-to-date
		}

		for _, condition := range deployment.Status.Conditions {
			if condition.Type == kapps.DeploymentProgressing &&
				condition.Status == kcore.ConditionTrue &&
				!condition.LastUpdateTime.IsZero() &&
				time.Now().After(condition.LastUpdateTime.Add(35*time.Second)) { // the metrics poll interval is 30 seconds, so 35 should be safe

				spec, err := hpaSpec(&deployment)
				if err != nil {
					errs = append(errs, err)
					continue
				}

				if _, err = config.K8s.CreateHPA(spec); err != nil {
					errs = append(errs, err)
					continue
				}
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
