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

	corev1 "k8s.io/api/core/v1"

	"github.com/cortexlabs/cortex/pkg/api/resource"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/parallel"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/operator/aws"
	ocontext "github.com/cortexlabs/cortex/pkg/operator/context"
	"github.com/cortexlabs/cortex/pkg/operator/k8s"
)

func uploadAPISavedStatus(savedStatus *resource.APISavedStatus) error {
	if isAPISavedStatusCached(savedStatus) {
		return nil
	}

	key := ocontext.StatusKey(savedStatus.ResourceID, savedStatus.WorkloadID, savedStatus.AppName)
	err := aws.UploadJSONToS3(savedStatus, key)
	if err != nil {
		return errors.Wrap(err, "upload api saved status", savedStatus.AppName, savedStatus.ResourceID, savedStatus.WorkloadID)
	}
	cacheAPISavedStatus(savedStatus)
	return nil
}

func uploadAPISavedStatuses(savedStatuses []*resource.APISavedStatus) error {
	fns := make([]func() error, len(savedStatuses))
	for i, savedStatus := range savedStatuses {
		fns[i] = uploadAPISavedStatusFunc(savedStatus)
	}
	return parallel.RunFirstErr(fns...)
}

func uploadAPISavedStatusFunc(savedStatus *resource.APISavedStatus) func() error {
	return func() error {
		return uploadAPISavedStatus(savedStatus)
	}
}

func getAPISavedStatus(resourceID string, workloadID string, appName string) (*resource.APISavedStatus, error) {
	if cachedSavedStatus, ok := getCachedAPISavedStatus(resourceID, workloadID, appName); ok {
		return cachedSavedStatus, nil
	}

	key := ocontext.StatusKey(resourceID, workloadID, appName)
	var savedStatus resource.APISavedStatus
	err := aws.ReadJSONFromS3(&savedStatus, key)
	if aws.IsNoSuchKeyErr(err) {
		cacheNilAPISavedStatus(resourceID, workloadID, appName)
		return nil, nil
	}
	if err != nil {
		return nil, errors.Wrap(err, "download api saved status", appName, resourceID, workloadID)
	}
	cacheAPISavedStatus(&savedStatus)
	return &savedStatus, nil
}

func calculateAPISavedStatuses(podList []corev1.Pod, appName string) ([]*resource.APISavedStatus, error) {
	podMap := make(map[string]map[string][]corev1.Pod)
	for _, pod := range podList {
		resourceID := pod.Labels["resourceID"]
		workloadID := pod.Labels["workloadID"]
		if _, ok := podMap[resourceID]; !ok {
			podMap[resourceID] = make(map[string][]corev1.Pod)
		}
		podMap[resourceID][workloadID] = append(podMap[resourceID][workloadID], pod)
	}

	var savedStatuses []*resource.APISavedStatus
	for resourceID := range podMap {
		for workloadID, pods := range podMap[resourceID] {
			savedStatus, err := getAPISavedStatus(resourceID, workloadID, appName)
			if err != nil {
				return nil, err
			}
			if savedStatus == nil {
				savedStatus = &resource.APISavedStatus{
					BaseSavedStatus: resource.BaseSavedStatus{
						ResourceID:   resourceID,
						ResourceType: resource.APIType,
						WorkloadID:   workloadID,
						AppName:      pods[0].Labels["appName"],
					},
					APIName: pods[0].Labels["apiName"],
				}
			}

			updateAPISavedStatusStartTime(savedStatus, pods)

			savedStatuses = append(savedStatuses, savedStatus)
		}
	}

	return savedStatuses, nil
}

func updateAPISavedStatusStartTime(savedStatus *resource.APISavedStatus, pods []corev1.Pod) {
	if savedStatus.Start != nil {
		return
	}

	for _, pod := range pods {
		podStatus := k8s.GetPodStatus(&pod)
		if podStatus != k8s.PodStatusRunning {
			continue
		}
		podStartTime := k8s.GetPodLastContainerStartTime(&pod)
		if podStartTime == nil {
			podStartTime = pointer.Time(time.Now())
		}
		if savedStatus.Start == nil || (*podStartTime).Before(*savedStatus.Start) {
			savedStatus.Start = podStartTime
		}
	}
}

func UpdateAPISavedStatuses(allPods []corev1.Pod) error {
	podMap := make(map[string][]corev1.Pod)
	for _, pod := range allPods {
		appName := pod.Labels["appName"]
		podMap[appName] = append(podMap[appName], pod)
	}

	var allSavedStatuses []*resource.APISavedStatus
	for appName, podList := range podMap {
		savedStatuses, err := calculateAPISavedStatuses(podList, appName)
		if err != nil {
			return err
		}
		allSavedStatuses = append(allSavedStatuses, savedStatuses...)
	}

	err := uploadAPISavedStatuses(allSavedStatuses)
	if err != nil {
		return err
	}

	err = updateFinishedAPISavedStatuses(allSavedStatuses)
	if err != nil {
		return err
	}

	return nil
}

func updateFinishedAPISavedStatuses(allSavedStatuses []*resource.APISavedStatus) error {
	staleSavedStatuses := getStaleAPISavedStatuses(allSavedStatuses)
	for _, savedStatus := range staleSavedStatuses {
		if savedStatus.End == nil {
			savedStatus.End = pointer.Time(time.Now())
		}
	}

	err := uploadAPISavedStatuses(staleSavedStatuses)
	if err != nil {
		return err
	}

	uncacheFinishedAPISavedStatuses(allSavedStatuses)
	return nil
}
