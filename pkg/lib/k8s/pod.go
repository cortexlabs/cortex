/*
Copyright 2021 Cortex Labs, Inc.

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
	"bytes"
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	kcore "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
	kscheme "k8s.io/client-go/kubernetes/scheme"
	kremotecommand "k8s.io/client-go/tools/remotecommand"
)

var _podTypeMeta = kmeta.TypeMeta{
	APIVersion: "v1",
	Kind:       "Pod",
}

// pod termination reasons
// https://github.com/kubernetes/kube-state-metrics/blob/master/docs/pod-metrics.md
const (
	ReasonEvicted   = "Evicted"
	ReasonOOMKilled = "OOMKilled"
	ReasonCompleted = "Completed"
)

type PodSpec struct {
	Name        string
	K8sPodSpec  kcore.PodSpec
	Labels      map[string]string
	Annotations map[string]string
}

type PodStatus string

const (
	PodStatusPending      PodStatus = "Pending"
	PodStatusCreating     PodStatus = "Creating"
	PodStatusNotReady     PodStatus = "NotReady"
	PodStatusReady        PodStatus = "Ready"
	PodStatusErrImagePull PodStatus = "ErrImagePull"
	PodStatusTerminating  PodStatus = "Terminating"
	PodStatusFailed       PodStatus = "Failed"
	PodStatusKilled       PodStatus = "Killed"
	PodStatusKilledOOM    PodStatus = "KilledOOM"
	PodStatusStalled      PodStatus = "Stalled"

	PodStatusSucceeded PodStatus = "Succeeded"

	PodStatusUnknown PodStatus = "Unknown"
)

var (
	_killStatuses = map[int32]bool{
		137: true, // SIGKILL
		143: true, // SIGTERM
		130: true, // SIGINT
		129: true, // SIGHUP
	}

	_evictedMemoryMessageRegex = regexp.MustCompile(`(?i)low\W+on\W+resource\W+memory`)

	// https://github.com/kubernetes/kubernetes/blob/master/pkg/kubelet/images/types.go#L27
	_imagePullErrorStrings = strset.New("ErrImagePull", "ImagePullBackOff", "RegistryUnavailable")

	// https://github.com/kubernetes/kubernetes/blob/9f47110aa29094ed2878cf1d85874cb59214664a/staging/src/k8s.io/api/core/v1/types.go#L76-L77
	_creatingReasons = strset.New("ContainerCreating", "PodInitializing")

	_waitForCreatingPodTimeout = time.Minute * 15
)

func Pod(spec *PodSpec) *kcore.Pod {
	pod := &kcore.Pod{
		TypeMeta: _podTypeMeta,
		ObjectMeta: kmeta.ObjectMeta{
			Name:        spec.Name,
			Labels:      spec.Labels,
			Annotations: spec.Annotations,
		},
		Spec: spec.K8sPodSpec,
	}
	return pod
}

func GetPodConditionOf(pod *kcore.Pod, podType kcore.PodConditionType) (*bool, *kcore.PodCondition) {
	if pod == nil {
		return nil, nil
	}

	var conditionState *bool
	var condition *kcore.PodCondition
	for i := range pod.Status.Conditions {
		if pod.Status.Conditions[i].Type == podType {
			if pod.Status.Conditions[i].Status == kcore.ConditionTrue {
				conditionState = pointer.Bool(true)
			}
			if pod.Status.Conditions[i].Status == kcore.ConditionFalse {
				conditionState = pointer.Bool(false)
			}
			condition = &pod.Status.Conditions[i]
			break
		}
	}
	return conditionState, condition
}

func (c *Client) CreatePod(pod *kcore.Pod) (*kcore.Pod, error) {
	pod.TypeMeta = _podTypeMeta
	pod, err := c.podClient.Create(context.Background(), pod, kmeta.CreateOptions{})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return pod, nil
}

func (c *Client) UpdatePod(pod *kcore.Pod) (*kcore.Pod, error) {
	pod.TypeMeta = _podTypeMeta
	pod, err := c.podClient.Update(context.Background(), pod, kmeta.UpdateOptions{})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return pod, nil
}

func (c *Client) ApplyPod(pod *kcore.Pod) (*kcore.Pod, error) {
	existing, err := c.GetPod(pod.Name)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return c.CreatePod(pod)
	}
	return c.UpdatePod(pod)
}

func IsPodReady(pod *kcore.Pod) bool {
	if GetPodStatus(pod) != PodStatusReady {
		return false
	}

	// TODO use the GetPodConditionOf func here
	for _, condition := range pod.Status.Conditions {
		if condition.Type == kcore.PodReady && condition.Status == kcore.ConditionTrue {
			return true
		}
	}

	return false
}

func IsPodStalled(pod *kcore.Pod) bool {
	if GetPodStatus(pod) != PodStatusPending {
		return false
	}

	// TODO use the GetPodConditionOf func here
	for _, condition := range pod.Status.Conditions {
		if condition.Type == kcore.PodScheduled && condition.Status == kcore.ConditionFalse && !condition.LastTransitionTime.Time.IsZero() && time.Since(condition.LastTransitionTime.Time) >= _waitForCreatingPodTimeout {
			fmt.Println(time.Since(condition.LastTransitionTime.Time), _waitForCreatingPodTimeout)
			return true
		}
	}

	return false
}

func GetPodReadyTime(pod *kcore.Pod) *time.Time {
	for i := range pod.Status.Conditions {
		condition := pod.Status.Conditions[i]

		if condition.Type == kcore.PodReady && condition.Status == kcore.ConditionTrue {
			if condition.LastTransitionTime.Time.IsZero() {
				return nil
			}
			return &condition.LastTransitionTime.Time
		}
	}

	return nil
}

func WasPodOOMKilled(pod *kcore.Pod) bool {
	if pod.Status.Reason == ReasonEvicted && _evictedMemoryMessageRegex.MatchString(pod.Status.Message) {
		return true
	}
	for _, containerStatus := range pod.Status.ContainerStatuses {
		var reason string
		if containerStatus.LastTerminationState.Terminated != nil {
			reason = containerStatus.LastTerminationState.Terminated.Reason
		} else if containerStatus.State.Terminated != nil {
			reason = containerStatus.State.Terminated.Reason
		}
		if reason == ReasonOOMKilled {
			return true
		}
	}

	return false
}

func GetPodStatus(pod *kcore.Pod) PodStatus {
	if pod == nil {
		return PodStatusUnknown
	}

	switch pod.Status.Phase {
	case kcore.PodPending:
		podConditionState, podCondition := GetPodConditionOf(pod, kcore.PodScheduled)
		if podConditionState != nil && !*podConditionState && !podCondition.LastTransitionTime.Time.IsZero() && time.Since(podCondition.LastTransitionTime.Time) >= _waitForCreatingPodTimeout {
			return PodStatusStalled
		}
		return PodStatusFromContainerStatuses(append(pod.Status.InitContainerStatuses, pod.Status.ContainerStatuses...))
	case kcore.PodSucceeded:
		return PodStatusSucceeded
	case kcore.PodFailed:
		if pod.Status.Reason == ReasonEvicted && _evictedMemoryMessageRegex.MatchString(pod.Status.Message) {
			return PodStatusKilledOOM
		}

		for _, containerStatus := range pod.Status.ContainerStatuses {
			var reason string
			var exitCode int32
			if containerStatus.LastTerminationState.Terminated != nil {
				reason = containerStatus.LastTerminationState.Terminated.Reason
				exitCode = containerStatus.LastTerminationState.Terminated.ExitCode
			} else if containerStatus.State.Terminated != nil {
				reason = containerStatus.State.Terminated.Reason
				exitCode = containerStatus.State.Terminated.ExitCode
			}
			if reason == ReasonOOMKilled {
				return PodStatusKilledOOM
			} else if _killStatuses[exitCode] {
				return PodStatusKilled
			}

		}
		return PodStatusFailed
	case kcore.PodRunning:
		if pod.ObjectMeta.DeletionTimestamp != nil {
			return PodStatusTerminating
		}

		podConditionState, _ := GetPodConditionOf(pod, kcore.PodReady)
		if podConditionState != nil && *podConditionState {
			return PodStatusReady
		}

		status := PodStatusFromContainerStatuses(pod.Status.ContainerStatuses)
		if status == PodStatusReady || status == PodStatusNotReady {
			return PodStatusNotReady
		}

		return status
	default:
		return PodStatusUnknown
	}
}

func PodStatusFromContainerStatuses(containerStatuses []kcore.ContainerStatus) PodStatus {
	numContainers := len(containerStatuses)
	numWaiting := 0
	numCreating := 0
	numNotReady := 0
	numReady := 0
	numSucceeded := 0
	numFailed := 0
	numKilled := 0
	numKilledOOM := 0

	if len(containerStatuses) == 0 {
		return PodStatusPending
	}
	for _, containerStatus := range containerStatuses {
		if containerStatus.State.Running != nil && containerStatus.Ready {
			numReady++
		} else if containerStatus.State.Running != nil && !containerStatus.Ready {
			numNotReady++
		} else if containerStatus.State.Terminated != nil {
			exitCode := containerStatus.State.Terminated.ExitCode
			reason := containerStatus.State.Terminated.Reason
			if reason == ReasonOOMKilled {
				numKilledOOM++
			} else if exitCode == 0 {
				numSucceeded++
			} else if _killStatuses[exitCode] {
				numKilled++
			} else {
				numFailed++
			}
		} else if containerStatus.LastTerminationState.Terminated != nil {
			exitCode := containerStatus.LastTerminationState.Terminated.ExitCode
			reason := containerStatus.LastTerminationState.Terminated.Reason
			if reason == ReasonOOMKilled {
				numKilledOOM++
			} else if exitCode == 0 {
				numSucceeded++
			} else if _killStatuses[exitCode] {
				numKilled++
			} else {
				numFailed++
			}
		} else if containerStatus.State.Waiting != nil && _imagePullErrorStrings.Has(containerStatus.State.Waiting.Reason) {
			return PodStatusErrImagePull
		} else if containerStatus.State.Waiting != nil && _creatingReasons.Has(containerStatus.State.Waiting.Reason) {
			numCreating++
		} else {
			// either containerStatus.State.Waiting != nil or all containerStatus.States are nil (which implies waiting)
			numWaiting++
		}
	}
	if numKilledOOM > 0 {
		return PodStatusKilledOOM
	} else if numKilled > 0 {
		return PodStatusKilled
	} else if numFailed > 0 {
		return PodStatusFailed
	} else if numWaiting > 0 {
		return PodStatusPending
	} else if numSucceeded == numContainers {
		return PodStatusSucceeded
	} else if numCreating > 0 {
		return PodStatusCreating
	} else if numNotReady > 0 {
		return PodStatusNotReady
	} else {
		return PodStatusReady
	}
}

func (c *Client) WaitForPodRunning(name string, numSeconds int) error {
	for true {
		pod, err := c.GetPod(name)
		if err != nil {
			return err
		}
		if pod != nil && pod.Status.Phase == kcore.PodRunning {
			return nil
		}
		time.Sleep(time.Duration(numSeconds) * time.Second)
	}
	return nil
}

func (c *Client) GetPod(name string) (*kcore.Pod, error) {
	pod, err := c.podClient.Get(context.Background(), name, kmeta.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, errors.WithStack(err)
	}
	pod.TypeMeta = _podTypeMeta
	return pod, nil
}

func (c *Client) DeletePod(name string) (bool, error) {
	err := c.podClient.Delete(context.Background(), name, _deleteOpts)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return false, nil
		}
		return false, errors.WithStack(err)
	}
	return true, nil
}

func (c *Client) ListPods(opts *kmeta.ListOptions) ([]kcore.Pod, error) {
	if opts == nil {
		opts = &kmeta.ListOptions{}
	}
	podList, err := c.podClient.List(context.Background(), *opts)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	for i := range podList.Items {
		podList.Items[i].TypeMeta = _podTypeMeta
	}
	return podList.Items, nil
}

func (c *Client) ListPodsByLabels(labels map[string]string) ([]kcore.Pod, error) {
	opts := &kmeta.ListOptions{
		LabelSelector: klabels.SelectorFromSet(labels).String(),
	}
	return c.ListPods(opts)
}

func (c *Client) ListPodsByLabel(labelKey string, labelValue string) ([]kcore.Pod, error) {
	return c.ListPodsByLabels(map[string]string{labelKey: labelValue})
}

func (c *Client) ListPodsWithLabelKeys(labelKeys ...string) ([]kcore.Pod, error) {
	opts := &kmeta.ListOptions{
		LabelSelector: LabelExistsSelector(labelKeys...),
	}
	return c.ListPods(opts)
}

func PodMap(pods []kcore.Pod) map[string]kcore.Pod {
	podMap := map[string]kcore.Pod{}
	for _, pod := range pods {
		podMap[pod.Name] = pod
	}
	return podMap
}

func PodComputesEqual(podSpec1, podSpec2 *kcore.PodSpec) bool {
	cpu1, mem1, gpu1, inf1 := TotalPodCompute(podSpec1)
	cpu2, mem2, gpu2, inf2 := TotalPodCompute(podSpec2)
	return cpu1.Equal(cpu2) && mem1.Equal(mem2) && gpu1 == gpu2 && inf1 == inf2
}

func TotalPodCompute(podSpec *kcore.PodSpec) (Quantity, Quantity, int64, int64) {
	totalCPU := Quantity{}
	totalMem := Quantity{}
	var totalGPU, totalInf int64

	if podSpec == nil {
		return totalCPU, totalMem, totalGPU, totalInf
	}

	for _, container := range podSpec.Containers {
		requests := container.Resources.Requests
		if len(requests) == 0 {
			continue
		}
		totalCPU.Add(requests[kcore.ResourceCPU])
		totalMem.Add(requests[kcore.ResourceMemory])
		if gpu, ok := requests["nvidia.com/gpu"]; ok {
			totalGPU += gpu.Value()
		}
		if inf, ok := requests["aws.amazon.com/neuron"]; ok {
			totalInf += inf.Value()
		}
	}

	return totalCPU, totalMem, totalGPU, totalInf
}

// Example of running a shell command: []string{"/bin/bash", "-c", "ps aux | grep my-proc"}
func (c *Client) Exec(podName string, containerName string, command []string) (string, error) {
	options := &kcore.PodExecOptions{
		Container: containerName,
		Command:   command,
		Stdin:     false,
		Stdout:    true,
		Stderr:    true,
		TTY:       true,
	}

	req := c.clientSet.CoreV1().RESTClient().Post().Namespace(c.Namespace).Resource("pods").Name(podName).SubResource("exec")
	req.VersionedParams(options, kscheme.ParameterCodec)

	exec, err := kremotecommand.NewSPDYExecutor(c.RestConfig, "POST", req.URL())
	if err != nil {
		return "", err
	}

	buf := &bytes.Buffer{}

	err = exec.Stream(kremotecommand.StreamOptions{
		Stdin:  nil,
		Stdout: buf,
		Stderr: nil, // TTY merges stdout and stderr
		Tty:    true,
	})
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}
