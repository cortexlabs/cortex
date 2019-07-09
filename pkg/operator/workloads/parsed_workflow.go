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

	argowf "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	kcore "k8s.io/api/core/v1"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/cortexlabs/cortex/pkg/lib/argo"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/operator/config"
)

type WorkflowItem struct {
	WorkloadID         string
	WorkloadType       string
	StartedAt          *time.Time
	FinishedAt         *time.Time
	ArgoPhase          *argowf.NodePhase
	DirectDependencies strset.Set
	AllDependencies    strset.Set
}

type ParsedWorkflow struct {
	Workloads map[string]*WorkflowItem // workloadID -> *WorkflowItem
	Wf        *argowf.Workflow
}

func parseWorkflow(wf *argowf.Workflow) (*ParsedWorkflow, error) {
	if wf == nil {
		return nil, nil
	}

	pWf := &ParsedWorkflow{
		Workloads: map[string]*WorkflowItem{},
		Wf:        wf,
	}

	for _, argoWfItem := range argo.ParseWorkflow(wf) {
		workloadID := argoWfItem.Labels["workloadID"]
		workloadType := argoWfItem.Labels["workloadType"]
		if workloadID == "" || workloadType == "" {
			continue
		}

		pWf.Workloads[workloadID] = &WorkflowItem{
			WorkloadID:         workloadID,
			WorkloadType:       workloadType,
			StartedAt:          argoWfItem.StartedAt(),
			FinishedAt:         argoWfItem.FinishedAt(),
			ArgoPhase:          argoWfItem.Phase(),
			DirectDependencies: argoWfItem.Dependencies(),
		}
	}

	for workloadID, wfItem := range pWf.Workloads {
		allDependencies, err := getAllDependencies(workloadID, pWf.Workloads)
		if err != nil {
			return nil, err
		}
		wfItem.AllDependencies = allDependencies
	}

	return pWf, nil
}

func getAllDependencies(workloadID string, workloads map[string]*WorkflowItem) (strset.Set, error) {
	wfItem, ok := workloads[workloadID]
	if !ok {
		return nil, errors.Wrap(ErrorNotFound(), "workload", workloadID)
	}
	allDependencies := strset.New()
	if len(wfItem.DirectDependencies) == 0 {
		return allDependencies, nil
	}
	for dependency := range wfItem.DirectDependencies {
		allDependencies.Add(dependency)
		subDependencies, err := getAllDependencies(dependency, workloads)
		if err != nil {
			return nil, err
		}
		allDependencies.Merge(subDependencies)
	}
	return allDependencies, nil
}

func getFailedArgoWorkloadIDs(appName string) (strset.Set, error) {
	failedArgoPods, err := config.Kubernetes.ListPods(&kmeta.ListOptions{
		FieldSelector: "status.phase=Failed",
		LabelSelector: k8s.LabelSelector(map[string]string{
			"appName": appName,
			"argo":    "true",
		}),
	})
	if err != nil {
		return nil, err
	}

	failedWorkloadIDs := strset.New()
	for _, pod := range failedArgoPods {
		failedWorkloadIDs.Add(pod.Labels["workloadID"])
	}
	return failedWorkloadIDs, nil
}

func getFailedArgoPodForWorkload(workloadID string, appName string) (*kcore.Pod, error) {
	failedArgoPods, err := config.Kubernetes.ListPods(&kmeta.ListOptions{
		FieldSelector: "status.phase=Failed",
		LabelSelector: k8s.LabelSelector(map[string]string{
			"appName":    appName,
			"workloadID": workloadID,
			"argo":       "true",
		}),
	})
	if err != nil {
		return nil, err
	}

	if len(failedArgoPods) == 0 {
		return nil, nil
	}

	return &failedArgoPods[0], nil
}
