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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/cortexlabs/cortex/pkg/api/context"
	"github.com/cortexlabs/cortex/pkg/api/resource"
	"github.com/cortexlabs/cortex/pkg/operator/aws"
	ocontext "github.com/cortexlabs/cortex/pkg/operator/context"
	"github.com/cortexlabs/cortex/pkg/utils/errors"
	"github.com/cortexlabs/cortex/pkg/utils/sets/strset"
	"github.com/cortexlabs/cortex/pkg/utils/util"
)

type WorkloadSpec struct {
	WorkloadID       string
	ResourceIDs      strset.Set
	Spec             metav1.Object
	K8sAction        string
	SuccessCondition string
	FailureCondition string
	WorkloadType     string
}

type SavedWorkloadSpec struct {
	AppName      string
	WorkloadID   string
	WorkloadType string
	Resources    map[string]*context.ResourceFields
}

func uploadWorkloadSpec(workloadSpec *WorkloadSpec, ctx *context.Context) error {
	if workloadSpec == nil {
		return nil
	}

	resources := make(map[string]*context.ResourceFields)
	for resourceID := range workloadSpec.ResourceIDs {
		resource := ctx.OneResourceByID(resourceID)
		resources[resourceID] = resource.GetResourceFields()
	}

	savedWorkloadSpec := SavedWorkloadSpec{
		AppName:      ctx.App.Name,
		WorkloadID:   workloadSpec.WorkloadID,
		WorkloadType: workloadSpec.WorkloadType,
		Resources:    resources,
	}

	key := ocontext.WorkloadSpecKey(savedWorkloadSpec.WorkloadID, ctx.App.Name)
	err := aws.UploadJSONToS3(savedWorkloadSpec, key)
	if err != nil {
		return errors.Wrap(err, "upload workload spec", ctx.App.Name, savedWorkloadSpec.WorkloadID)
	}
	return nil
}

func getSavedWorkloadSpec(workloadID string, appName string) (*SavedWorkloadSpec, error) {
	key := ocontext.WorkloadSpecKey(workloadID, appName)
	var savedWorkloadSpec SavedWorkloadSpec
	err := aws.ReadJSONFromS3(&savedWorkloadSpec, key)
	if aws.IsNoSuchKeyErr(err) {
		return nil, nil
	}
	if err != nil {
		return nil, errors.Wrap(err, "download workload spec", appName, workloadID)
	}
	return &savedWorkloadSpec, nil
}

func UpdateDataWorkflowErrors(failedPods []corev1.Pod) error {
	checkedWorkloadIDs := strset.New()
	nowTime := util.TimeNowPtr()

	for _, pod := range failedPods {
		appName, ok := pod.Labels["appName"]
		if !ok {
			continue
		}
		workloadID, ok := pod.Labels["workloadID"]
		if !ok {
			continue
		}

		if pod.Labels["workloadType"] == WorkloadTypeAPI {
			continue
		}

		if checkedWorkloadIDs.Has(workloadID) {
			continue
		}
		checkedWorkloadIDs.Add(workloadID)

		savedWorkloadSpec, err := getSavedWorkloadSpec(workloadID, appName)
		if err != nil {
			return err
		}

		resourceWorkloadIDs := make(map[string]string, len(savedWorkloadSpec.Resources))
		for _, resource := range savedWorkloadSpec.Resources {
			resourceWorkloadIDs[resource.ID] = workloadID
		}

		savedStatuses, err := getDataSavedStatuses(resourceWorkloadIDs, appName)
		if err != nil {
			return err
		}

		var savedStatusesToUpload []*resource.DataSavedStatus
		for resourceID, res := range savedWorkloadSpec.Resources {
			savedStatus := savedStatuses[resourceID]

			if savedStatus == nil {
				savedStatus = &resource.DataSavedStatus{
					BaseSavedStatus: resource.BaseSavedStatus{
						ResourceID:   resourceID,
						ResourceType: res.ResourceType,
						WorkloadID:   workloadID,
						AppName:      appName,
					},
				}
			}

			if savedStatus.End == nil {
				savedStatus.End = nowTime
				if savedStatus.Start == nil {
					savedStatus.Start = nowTime
				}
				savedStatus.ExitCode = resource.ExitCodeDataFailed
				savedStatusesToUpload = append(savedStatusesToUpload, savedStatus)
			}
		}

		err = uploadDataSavedStatuses(savedStatusesToUpload)
		if err != nil {
			return err
		}
	}
	return nil
}
