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
	"bufio"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	kcore "k8s.io/api/core/v1"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/operator/config"
)

const (
	socketWriteDeadlineWait = 10 * time.Second
	socketCloseGracePeriod  = 10 * time.Second
	socketMaxMessageSize    = 8192

	pendingPodCheckInterval = 1 * time.Second
	newPodCheckInterval     = 5 * time.Second
	firstPodCheckInterval   = 500 * time.Millisecond
	maxParallelPodLogging   = 5
	initLogTailLines        = 100
)

func ReadLogs(appName string, podSearchLabels map[string]string, socket *websocket.Conn) {
	wrotePending := false

	for true {
		pods, err := config.Kubernetes.ListPodsByLabels(podSearchLabels)
		if err != nil {
			writeSocket(err.Error(), socket)
			return
		}

		if len(pods) > 0 {
			if len(pods) > maxParallelPodLogging {
				if !writeSocket(fmt.Sprintf("\n%d pods available, streaming logs for %d of them:", len(pods), maxParallelPodLogging), socket) {
					return
				}
			}

			podMap := make(map[k8s.PodStatus][]kcore.Pod)
			for _, pod := range pods {
				podStatus := k8s.GetPodStatus(&pod)
				if len(podMap[podStatus]) < maxParallelPodLogging {
					podMap[podStatus] = append(podMap[podStatus], pod)
				}
			}

			switch {
			case len(podMap[k8s.PodStatusSucceeded]) > 0:
				getKubectlLogs(podMap[k8s.PodStatusSucceeded], wrotePending, socket)
			case len(podMap[k8s.PodStatusRunning]) > 0:
				getKubectlLogs(podMap[k8s.PodStatusRunning], wrotePending, socket)
			case len(podMap[k8s.PodStatusPending]) > 0:
				getKubectlLogs(podMap[k8s.PodStatusPending], wrotePending, socket)
			case len(podMap[k8s.PodStatusKilled]) > 0:
				getKubectlLogs(podMap[k8s.PodStatusKilled], wrotePending, socket)
			case len(podMap[k8s.PodStatusKilledOOM]) > 0:
				getKubectlLogs(podMap[k8s.PodStatusKilledOOM], wrotePending, socket)
			case len(podMap[k8s.PodStatusFailed]) > 0:
				getKubectlLogs(podMap[k8s.PodStatusFailed], wrotePending, socket)
			case len(podMap[k8s.PodStatusTerminating]) > 0:
				getKubectlLogs(podMap[k8s.PodStatusTerminating], wrotePending, socket)
			case len(podMap[k8s.PodStatusUnknown]) > 0:
				getKubectlLogs(podMap[k8s.PodStatusUnknown], wrotePending, socket)
			default: // unexpected
				if len(pods) > maxParallelPodLogging {
					pods = pods[:maxParallelPodLogging]
				}
				getKubectlLogs(pods, wrotePending, socket)
			}
			return
		}

		// WorkloadID is present when logging job pods and missing for APIs. Only go to cloudwatch for job pods.
		if workloadID, ok := podSearchLabels["workloadID"]; ok {
			isEnded, err := IsWorkloadEnded(appName, workloadID)
			if err != nil {
				writeSocket(err.Error(), socket)
				return
			}

			if isEnded {
				getCloudWatchLogs(workloadID, socket)
				return
			}
		}

		if !wrotePending {
			if !writeSocket("\nPending...", socket) {
				return
			}
			wrotePending = true
		}

		time.Sleep(pendingPodCheckInterval)
	}
}

func getKubectlLogs(pods []kcore.Pod, wrotePending bool, socket *websocket.Conn) {
	if !wrotePending {
		isAllPending := true
		for _, pod := range pods {
			if k8s.GetPodStatus(&pod) != k8s.PodStatusPending {
				isAllPending = false
				break
			}
		}

		if isAllPending {
			if !writeSocket("\nPending...", socket) {
				return
			}
			wrotePending = true
		}
	}

	inr, inw, err := os.Pipe()
	if err != nil {
		errors.Panic(err, "logs", "kubectl", "os.pipe")
	}
	defer inw.Close()
	defer inr.Close()

	podCheckCancel := make(chan struct{})
	defer close(podCheckCancel)

	go podCheck(podCheckCancel, socket, pods, wrotePending, inr)
	pumpStdin(socket, inw)
	podCheckCancel <- struct{}{}
}

type LogKey struct {
	PodName       string
	ContainerName string
	RestartCount  int32
}

func (l LogKey) String() string {
	return fmt.Sprintf("%s,%s,%d", l.PodName, l.ContainerName, l.RestartCount)
}

func (l LogKey) KubeIdentifier() string {
	return fmt.Sprintf("%s %s", l.PodName, l.ContainerName)
}

func StringToLogKey(str string) LogKey {
	split := strings.Split(str, ",")
	restartCount, _ := s.ParseInt32(split[2])
	return LogKey{PodName: split[0], ContainerName: split[1], RestartCount: restartCount}
}

func GetLogKey(pod kcore.Pod, status kcore.ContainerStatus) LogKey {
	return LogKey{PodName: pod.Name, ContainerName: status.Name, RestartCount: status.RestartCount}
}

func CurrentLoggingProcesses(logProcesses map[string]*os.Process) strset.Set {
	set := strset.New()
	for identifier := range logProcesses {
		set.Add(identifier)
	}
	return set
}

func GetLogKeys(pod kcore.Pod) strset.Set {
	containerStatuses := append(pod.Status.InitContainerStatuses, pod.Status.ContainerStatuses...)
	logKeys := strset.New()
	for _, status := range containerStatuses {
		if status.State.Terminated != nil && (status.State.Terminated.ExitCode != 0 || status.State.Terminated.StartedAt.After(time.Now().Add(-newPodCheckInterval))) {
			logKeys.Add(GetLogKey(pod, status).String())
		} else if status.State.Running != nil {
			logKeys.Add(GetLogKey(pod, status).String())
		}
	}

	return logKeys
}

func createKubectlProcess(logKey LogKey, attrs *os.ProcAttr) (*os.Process, error) {
	cmdPath := "/bin/bash"

	kubectlArgs := []string{"kubectl", "-n=" + config.Cortex.Namespace, "logs", "--follow=true", logKey.PodName, logKey.ContainerName}

	identifier := logKey.KubeIdentifier()

	kubectlArgs = append(kubectlArgs, fmt.Sprintf("--tail=%d", initLogTailLines))
	labelLog := fmt.Sprintf(" | while read -r; do echo \"[%s] $REPLY\" | tail -n +1; done", identifier)
	kubectlArgsCmd := strings.Join(kubectlArgs, " ")
	bashArgs := []string{"/bin/bash", "-c", kubectlArgsCmd + labelLog}
	process, err := os.StartProcess(cmdPath, bashArgs, attrs)
	if err != nil {
		return nil, errors.Wrap(err, strings.Join(bashArgs, " "))
	}
	return process, nil
}

func podCheck(podCheckCancel chan struct{}, socket *websocket.Conn, initialPodList []kcore.Pod, wrotePending bool, inr *os.File) {
	timer := time.NewTimer(0)
	defer timer.Stop()

	processMap := make(map[string]*os.Process)
	defer deleteProcesses(processMap)
	labels := initialPodList[0].GetLabels()
	podSearchLabels := map[string]string{
		"appName":      labels["appName"],
		"workloadType": labels["workloadType"],
		"userFacing":   "true",
	}

	if labels["workloadType"] == workloadTypeAPI {
		podSearchLabels["apiName"] = labels["apiName"]
	} else {
		podSearchLabels["workloadID"] = labels["workloadID"]
	}

	outr, outw, err := os.Pipe()
	if err != nil {
		errors.Panic(err, "logs", "kubectl", "os.pipe")
	}
	defer outw.Close()
	defer outr.Close()

	socketWriterError := make(chan error, 1)
	defer close(socketWriterError)

	go pumpStdout(socket, socketWriterError, outr, true)

	for {
		select {
		case <-podCheckCancel:
			return
		case <-timer.C:
			pods, err := config.Kubernetes.ListPodsByLabels(podSearchLabels)

			if err != nil {
				socketWriterError <- errors.Wrap(err, "pod check")
				return
			}

			latestRunningPodsMap := make(map[string]kcore.Pod)
			latestRunningPods := strset.New()
			for _, pod := range pods {
				if k8s.GetPodStatus(&pod) != k8s.PodStatusPending {
					latestRunningPods.Add(pod.GetName())
					latestRunningPodsMap[pod.GetName()] = pod
				}
			}

			runningLogProcesses := CurrentLoggingProcesses(processMap)

			sortedPods := latestRunningPods.Slice()
			sort.Slice(sortedPods, func(i, j int) bool {
				return latestRunningPodsMap[sortedPods[i]].CreationTimestamp.After(latestRunningPodsMap[sortedPods[j]].CreationTimestamp.Time)
			})

			expectedLogProcesses := strset.New()
			for i, podName := range sortedPods {
				if i >= maxParallelPodLogging {
					break
				}
				expectedLogProcesses.Merge(GetLogKeys(latestRunningPodsMap[podName]))
			}

			processesToDelete := strset.Difference(runningLogProcesses, expectedLogProcesses)
			processesToAdd := strset.Difference(expectedLogProcesses, runningLogProcesses)

			if wrotePending && len(latestRunningPods) > 0 {
				if !writeSocket("Streaming logs:", socket) {
					return
				}
				wrotePending = false
			}

			for logProcess := range processesToAdd {
				process, err := createKubectlProcess(StringToLogKey(logProcess), &os.ProcAttr{
					Files: []*os.File{inr, outw, outw},
				})
				if err != nil {
					socketWriterError <- err
					return
				}
				processMap[logProcess] = process
			}

			deleteMap := make(map[string]*os.Process, len(processesToDelete))

			for podName := range processesToDelete {
				deleteMap[podName] = processMap[podName]
				delete(processMap, podName)
			}

			go deleteProcesses(deleteMap)

			if len(processMap) == 0 {
				timer.Reset(firstPodCheckInterval)
			} else {
				timer.Reset(newPodCheckInterval)
			}
		}
	}
}

func deleteProcesses(processMap map[string]*os.Process) {
	for _, process := range processMap {
		process.Signal(os.Interrupt)
	}
	time.Sleep(3 * time.Second)
	for _, process := range processMap {
		process.Signal(os.Kill)
	}
}

func getCloudWatchLogs(prefix string, socket *websocket.Conn) {
	logs, err := config.AWS.GetLogs(prefix, config.Cortex.LogGroup)
	if err != nil {
		config.Telemetry.ReportError(err)
		errors.PrintError(err)
	}

	var logsReader *strings.Reader
	if err != nil {
		logsReader = strings.NewReader(err.Error())
	} else if logs == "" {
		logsReader = strings.NewReader(prefix + " not found")
	} else {
		logsReader = strings.NewReader(logs)
	}

	socketWriterError := make(chan error)
	defer close(socketWriterError)
	go pumpStdout(socket, socketWriterError, logsReader, false)

	inr, inw, err := os.Pipe()
	if err != nil {
		errors.Panic(err, "logs", "cloudwatch", "os.pipe")
	}
	defer inr.Close()
	defer inw.Close()

	pumpStdin(socket, inw)
}

func pumpStdin(socket *websocket.Conn, writer io.Writer) {
	socket.SetReadLimit(socketMaxMessageSize)
	for {
		_, message, err := socket.ReadMessage()
		if err != nil {
			break
		}

		message = append(message, '\n')
		_, err = writer.Write(message)
		if err != nil {
			break
		}
	}
}

func pumpStdout(socket *websocket.Conn, socketWriterError chan error, reader io.Reader, checkForLastLog bool) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		socket.SetWriteDeadline(time.Now().Add(socketWriteDeadlineWait))
		logBytes := scanner.Bytes()
		isLastLog := false

		if logBytes != nil {
			if !writeSocketBytes(logBytes, socket) {
				break
			}
		}
		if isLastLog && checkForLastLog {
			break
		}
	}

	select {
	case err := <-socketWriterError:
		if err != nil {
			writeSocket(err.Error(), socket)
		}
	default:
	}

	socket.SetWriteDeadline(time.Now().Add(socketWriteDeadlineWait))
	socket.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	time.Sleep(socketCloseGracePeriod)
	socket.Close()
}

func stopProcess(process *os.Process) {
	process.Signal(os.Interrupt)
	time.Sleep(5 * time.Second)
	process.Signal(os.Kill)
}

func writeSocket(message string, socket *websocket.Conn) bool {
	return writeSocketBytes([]byte(message), socket)
}

func writeSocketBytes(message []byte, socket *websocket.Conn) bool {
	err := socket.WriteMessage(websocket.TextMessage, message)
	if err != nil {
		return false
	}
	return true
}
