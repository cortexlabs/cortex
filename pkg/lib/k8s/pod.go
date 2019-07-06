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

package k8s

import (
	"time"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	libtime "github.com/cortexlabs/cortex/pkg/lib/time"
)

var podTypeMeta = metav1.TypeMeta{
	APIVersion: "v1",
	Kind:       "Pod",
}

type PodStatus string

const (
	PodStatusUnknown     PodStatus = "Unknown"
	PodStatusPending     PodStatus = "Pending"
	PodStatusRunning     PodStatus = "Running"
	PodStatusTerminating PodStatus = "Terminating"
	PodStatusSucceeded   PodStatus = "Succeeded"
	PodStatusFailed      PodStatus = "Failed"
	PodStatusKilled      PodStatus = "Killed"
	PodStatusKilledOOM   PodStatus = "Out of Memory"
)

var killStatuses = map[int32]bool{
	137: true, // SIGKILL
	143: true, // SIGTERM
	130: true, // SIGINT
	129: true, // SIGHUP
}

type PodSpec struct {
	Name       string
	Namespace  string
	K8sPodSpec corev1.PodSpec
	Labels     map[string]string
}

func Pod(spec *PodSpec) *corev1.Pod {
	if spec.Namespace == "" {
		spec.Namespace = "default"
	}
	pod := &corev1.Pod{
		TypeMeta: podTypeMeta,
		ObjectMeta: metav1.ObjectMeta{
			Name:      spec.Name,
			Namespace: spec.Namespace,
			Labels:    spec.Labels,
		},
		Spec: spec.K8sPodSpec,
	}
	return pod
}

func (c *Client) CreatePod(spec *PodSpec) (*corev1.Pod, error) {
	pod, err := c.podClient.Create(Pod(spec))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return pod, nil
}

func (c *Client) UpdatePod(pod *corev1.Pod) (*corev1.Pod, error) {
	pod, err := c.podClient.Update(pod)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return pod, nil
}

func GetPodLastContainerStartTime(pod *corev1.Pod) *time.Time {
	var startTime *time.Time
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if containerStatus.State.Running == nil {
			return nil
		}
		containerStartTime := containerStatus.State.Running.StartedAt.Time
		if startTime == nil || containerStartTime.After(*startTime) {
			startTime = &containerStartTime
		}
	}
	return startTime
}

func GetPodStatus(pod *corev1.Pod) PodStatus {
	if pod == nil {
		return PodStatusUnknown
	}

	switch pod.Status.Phase {
	case corev1.PodPending:
		return PodStatusPending
	case corev1.PodSucceeded:
		return PodStatusSucceeded
	case corev1.PodFailed:
		for _, containerStatus := range pod.Status.ContainerStatuses {
			if containerStatus.LastTerminationState.Terminated != nil {
				exitCode := containerStatus.LastTerminationState.Terminated.ExitCode
				if exitCode == 137 {
					return PodStatusKilledOOM
				}
				if killStatuses[exitCode] {
					return PodStatusKilled
				}
			} else if containerStatus.State.Terminated != nil {
				exitCode := containerStatus.State.Terminated.ExitCode
				if exitCode == 137 {
					return PodStatusKilledOOM
				}
				if killStatuses[exitCode] {
					return PodStatusKilled
				}
			}
		}
		return PodStatusFailed
	case corev1.PodRunning:
		if pod.ObjectMeta.DeletionTimestamp != nil {
			return PodStatusTerminating
		}

		numContainers := len(pod.Status.ContainerStatuses)
		numWaiting := 0
		numRunning := 0
		numSucceeded := 0
		numFailed := 0
		numKilled := 0
		for _, containerStatus := range pod.Status.ContainerStatuses {
			if containerStatus.LastTerminationState.Terminated != nil {
				exitCode := containerStatus.LastTerminationState.Terminated.ExitCode
				if exitCode == 0 {
					numSucceeded++
				} else if killStatuses[exitCode] {
					numKilled++
				} else {
					numFailed++
				}
			} else if containerStatus.State.Waiting != nil {
				numWaiting++
			} else if containerStatus.State.Running != nil {
				if containerStatus.Ready {
					numRunning++
				} else {
					numWaiting++
				}
			} else if containerStatus.State.Terminated != nil {
				exitCode := containerStatus.State.Terminated.ExitCode
				if exitCode == 0 {
					numSucceeded++
				} else if killStatuses[exitCode] {
					numKilled++
				} else {
					numFailed++
				}
			} else {
				return PodStatusUnknown
			}
		}
		if numKilled > 0 {
			return PodStatusKilled
		} else if numFailed > 0 {
			return PodStatusFailed
		} else if numWaiting > 0 {
			return PodStatusPending
		} else if numSucceeded == numContainers {
			return PodStatusSucceeded
		} else {
			return PodStatusRunning
		}
	default:
		return PodStatusUnknown
	}
}

func (c *Client) WaitForPodRunning(name string, numSeconds int) error {
	for true {
		pod, err := c.GetPod(name)
		if err != nil {
			return err
		}
		if pod != nil && pod.Status.Phase == corev1.PodRunning {
			return nil
		}
		time.Sleep(time.Duration(numSeconds) * time.Second)
	}
	return nil
}

func (c *Client) GetPod(name string) (*corev1.Pod, error) {
	pod, err := c.podClient.Get(name, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, errors.WithStack(err)
	}
	pod.TypeMeta = podTypeMeta
	return pod, nil
}

func (c *Client) DeletePod(name string) (bool, error) {
	err := c.podClient.Delete(name, deleteOpts)
	if k8serrors.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, errors.WithStack(err)
	}
	return true, nil
}

func (c *Client) PodExists(name string) (bool, error) {
	pod, err := c.GetPod(name)
	if err != nil {
		return false, err
	}
	return pod != nil, nil
}

func (c *Client) ListPods(opts *metav1.ListOptions) ([]corev1.Pod, error) {
	if opts == nil {
		opts = &metav1.ListOptions{}
	}
	podList, err := c.podClient.List(*opts)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	for i := range podList.Items {
		podList.Items[i].TypeMeta = podTypeMeta
	}
	return podList.Items, nil
}

func (c *Client) ListPodsByLabels(labels map[string]string) ([]corev1.Pod, error) {
	opts := &metav1.ListOptions{
		LabelSelector: LabelSelector(labels),
	}
	return c.ListPods(opts)
}

func (c *Client) ListPodsByLabel(labelKey string, labelValue string) ([]corev1.Pod, error) {
	return c.ListPodsByLabels(map[string]string{labelKey: labelValue})
}

func PodMap(pods []corev1.Pod) map[string]corev1.Pod {
	podMap := map[string]corev1.Pod{}
	for _, pod := range pods {
		podMap[pod.Name] = pod
	}
	return podMap
}

func (c *Client) StalledPods() ([]corev1.Pod, error) {
	var stalledPods []corev1.Pod

	pods, err := c.ListPods(&metav1.ListOptions{
		FieldSelector: "status.phase=Pending",
	})
	if err != nil {
		return nil, err
	}
	for _, pod := range pods {
		if !libtime.OlderThanSeconds(pod.CreationTimestamp.Time, 3*60) {
			continue
		}
		stalledPods = append(stalledPods, pod)
	}

	return stalledPods, nil
}
