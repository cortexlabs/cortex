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
	"regexp"
	"time"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
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

type PodStatus string

const (
	PodStatusUnknown      PodStatus = "Unknown"
	PodStatusPending      PodStatus = "Pending"
	PodStatusInitializing PodStatus = "Initializing"
	PodStatusRunning      PodStatus = "Running"
	PodStatusErrImagePull PodStatus = "Image pull error"
	PodStatusTerminating  PodStatus = "Terminating"
	PodStatusSucceeded    PodStatus = "Succeeded"
	PodStatusFailed       PodStatus = "Failed"
	PodStatusKilled       PodStatus = "Killed"
	PodStatusKilledOOM    PodStatus = "Out of Memory"
)

var _killStatuses = map[int32]bool{
	137: true, // SIGKILL
	143: true, // SIGTERM
	130: true, // SIGINT
	129: true, // SIGHUP
}

// https://github.com/kubernetes/kubernetes/blob/master/pkg/kubelet/images/types.go#L27
var _imagePullErrorStrings = strset.New("ErrImagePull", "ImagePullBackOff", "RegistryUnavailable")

type PodSpec struct {
	Name        string
	K8sPodSpec  kcore.PodSpec
	Labels      map[string]string
	Annotations map[string]string
}

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
	if GetPodStatus(pod) != PodStatusRunning {
		return false
	}

	for _, condition := range pod.Status.Conditions {
		if condition.Type == "Ready" && condition.Status == kcore.ConditionTrue {
			return true
		}
	}

	return false
}

func GetPodReadyTime(pod *kcore.Pod) *time.Time {
	for i := range pod.Status.Conditions {
		condition := pod.Status.Conditions[i]

		if condition.Type == "Ready" && condition.Status == kcore.ConditionTrue {
			if condition.LastTransitionTime.Time.IsZero() {
				return nil
			}
			return &condition.LastTransitionTime.Time
		}
	}

	return nil
}

var _evictedMemoryMessageRegex = regexp.MustCompile(`(?i)low\W+on\W+resource\W+memory`)

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
		initPodStatus := PodStatusFromContainerStatuses(pod.Status.InitContainerStatuses)
		if initPodStatus == PodStatusRunning {
			return PodStatusInitializing
		}
		allPodStatus := PodStatusFromContainerStatuses(append(pod.Status.InitContainerStatuses, pod.Status.ContainerStatuses...))
		if allPodStatus == PodStatusErrImagePull {
			return PodStatusErrImagePull
		}
		return PodStatusPending
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

		return PodStatusFromContainerStatuses(pod.Status.ContainerStatuses)
	default:
		return PodStatusUnknown
	}
}

func PodStatusFromContainerStatuses(containerStatuses []kcore.ContainerStatus) PodStatus {
	numContainers := len(containerStatuses)
	numWaiting := 0
	numRunning := 0
	numSucceeded := 0
	numFailed := 0
	numKilled := 0
	numKilledOOM := 0

	if len(containerStatuses) == 0 {
		return PodStatusPending
	}
	for _, containerStatus := range containerStatuses {
		if containerStatus.State.Running != nil && containerStatus.Ready == true {
			numRunning++
		} else if containerStatus.State.Running != nil && containerStatus.RestartCount == 0 {
			numRunning++
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
	} else {
		return PodStatusRunning
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

	req := c.clientset.CoreV1().RESTClient().Post().Namespace(c.Namespace).Resource("pods").Name(podName).SubResource("exec")
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
