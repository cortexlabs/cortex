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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	kcore "k8s.io/api/core/v1"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/operator/config"
)

const (
	writeWait             = 10 * time.Second
	closeGracePeriod      = 10 * time.Second
	maxMessageSize        = 8192
	podCheckInterval      = 5 * time.Second
	maxParallelPodLogging = 5
)

func ReadLogs(appName string, workloadID string, verbose bool, socket *websocket.Conn) {
	wrotePending := false

	for true {
		pods, err := config.Kubernetes.ListPodsByLabels(map[string]string{
			"appName":    appName,
			"workloadID": workloadID,
			"userFacing": "true",
		})
		if err != nil {
			writeSocket(err.Error(), socket)
			return
		}

		if len(pods) > 0 {
			if len(pods) > maxParallelPodLogging {
				if !writeSocket(fmt.Sprintf("%d pods available, streaming logs for %d of them:", len(pods), maxParallelPodLogging), socket) {
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
				getKubectlLogs(podMap[k8s.PodStatusSucceeded], verbose, wrotePending, false, socket)
			case len(podMap[k8s.PodStatusRunning]) > 0:
				getKubectlLogs(podMap[k8s.PodStatusRunning], verbose, wrotePending, false, socket)
			case len(podMap[k8s.PodStatusPending]) > 0:
				getKubectlLogs(podMap[k8s.PodStatusPending], verbose, wrotePending, false, socket)
			case len(podMap[k8s.PodStatusKilled]) > 0:
				getKubectlLogs(podMap[k8s.PodStatusKilled], verbose, wrotePending, false, socket)
			case len(podMap[k8s.PodStatusKilledOOM]) > 0:
				getKubectlLogs(podMap[k8s.PodStatusKilledOOM], verbose, wrotePending, false, socket)
			case len(podMap[k8s.PodStatusFailed]) > 0:
				previous := false
				if pods[0].Labels["workloadType"] == WorkloadTypeAPI {
					previous = true
				}
				getKubectlLogs(podMap[k8s.PodStatusFailed], verbose, wrotePending, previous, socket)
			case len(podMap[k8s.PodStatusTerminating]) > 0:
				getKubectlLogs(podMap[k8s.PodStatusTerminating], verbose, wrotePending, false, socket)
			case len(podMap[k8s.PodStatusUnknown]) > 0:
				getKubectlLogs(podMap[k8s.PodStatusUnknown], verbose, wrotePending, false, socket)
			default: // unexpected
				if len(pods) > maxParallelPodLogging {
					pods = pods[:maxParallelPodLogging]
				}
				getKubectlLogs(pods, verbose, wrotePending, false, socket)
			}
			return
		}

		wf, _ := GetWorkflow(appName)
		pWf, _ := parseWorkflow(wf)
		if pWf == nil || pWf.Workloads[workloadID] == nil {
			logPrefix, err := getSavedLogPrefix(workloadID, appName, true)
			if err != nil {
				writeSocket(err.Error(), socket)
				return
			}
			if logPrefix == "" {
				logPrefix = workloadID
			}
			getCloudWatchLogs(logPrefix, verbose, socket)
			return
		}

		failedArgoPod, err := getFailedArgoPodForWorkload(workloadID, appName)
		if err != nil {
			writeSocket(err.Error(), socket)
			return
		}
		if failedArgoPod != nil {
			if !writeSocket("\nFailed to start:\n", socket) {
				return
			}
			getKubectlLogs([]kcore.Pod{*failedArgoPod}, true, false, false, socket)
			return
		}

		if !wrotePending {
			if !writeSocket("\nPending", socket) {
				return
			}
			wrotePending = true
		}

		time.Sleep(time.Duration(userFacingCheckInterval) * time.Second)
	}
}

func getKubectlLogs(pods []kcore.Pod, verbose bool, wrotePending bool, previous bool, socket *websocket.Conn) {
	isAllPending := true
	for _, pod := range pods {
		if k8s.GetPodStatus(&pod) != k8s.PodStatusPending {
			isAllPending = false
			break
		}
	}

	if isAllPending {
		if !writeSocket("\nPending", socket) {
			return
		}
	}

	outr, outw, err := os.Pipe()
	if err != nil {
		errors.Panic(err, "logs", "kubectl", "os.pipe")
	}
	defer outr.Close()

	inr, inw, err := os.Pipe()
	if err != nil {
		errors.Panic(err, "logs", "kubectl", "os.pipe")
	}
	defer inw.Close()

	podCheckCancel := make(chan struct{})
	defer close(podCheckCancel)
	socketWriterError := make(chan error, 1)
	go podCheck(podCheckCancel, socketWriterError, pods, previous, inr, outw)
	go pumpStdout(socket, socketWriterError, outr, verbose, true)
	pumpStdin(socket, inw)
	podCheckCancel <- struct{}{}
}

func startKubectlProcess(pod kcore.Pod, previous bool, attrs *os.ProcAttr) (*os.Process, error) {
	cmdPath := "/bin/bash"

	kubectlArgs := []string{"kubectl", "-n=" + config.Cortex.Namespace, "logs", "--follow=true"}
	if previous {
		kubectlArgs = append(kubectlArgs, "--previous")
	}

	identifier := pod.Name
	kubectlArgs = append(kubectlArgs, pod.Name)
	if pod.Labels["workloadType"] == WorkloadTypeAPI && pod.Labels["userFacing"] == "true" {
		kubectlArgs = append(kubectlArgs, apiContainerName)
		kubectlArgs = append(kubectlArgs, "--since="+podCheckInterval.String())
		identifier += " " + apiContainerName
	}

	labelLog := fmt.Sprintf(" | while read -r; do echo \"[%s] $REPLY \" | tail -n +1; done", identifier)
	kubectlCmd := strings.Join(kubectlArgs, " ")
	bashArgs := []string{"/bin/bash", "-c", kubectlCmd + labelLog}
	process, err := os.StartProcess(cmdPath, bashArgs, attrs)
	if err != nil {
		return nil, errors.Wrap(err, strings.Join(bashArgs, " "))
	}

	return process, nil
}

func podCheck(podCheckChan chan struct{}, socketWriterError chan error, initialPodList []kcore.Pod, previous bool, inr *os.File, outw *os.File) {
	defer inr.Close()
	defer outw.Close()

	timer := time.NewTimer(0)
	defer timer.Stop()

	processMap := make(map[string]*os.Process)
	defer deleteProcesses(processMap)
	labels := initialPodList[0].GetLabels()
	appName := labels["appName"]
	workloadID := labels["workloadID"]

	for {
		select {
		case <-podCheckChan:
			return
		case <-timer.C:
			pods, err := config.Kubernetes.ListPodsByLabels(map[string]string{
				"appName":    appName,
				"workloadID": workloadID,
				"userFacing": "true",
			})

			if err != nil {
				socketWriterError <- errors.Wrap(err, "pod check")
				return
			}

			runningPodsMap := make(map[string]kcore.Pod)
			latestRunningPods := strset.New()
			for _, pod := range pods {
				if k8s.GetPodStatus(&pod) != k8s.PodStatusPending {
					latestRunningPods.Add(pod.GetName())
					runningPodsMap[pod.GetName()] = pod
				}
			}

			runningPods := strset.New()

			for podName := range processMap {
				runningPods.Add(podName)
			}

			newPods := strset.Difference(latestRunningPods, runningPods)
			podsToDelete := strset.Difference(runningPods, latestRunningPods)
			podsToKeep := strset.Intersection(runningPods, latestRunningPods)

			// Prioritize adding running pods
			podsToAdd := []string{}
			podsToAddP2 := []string{}
			for podName := range newPods {
				pod := runningPodsMap[podName]
				if k8s.GetPodStatus(&pod) == k8s.PodStatusRunning {
					podsToAdd = append(podsToAdd, podName)
				} else {
					podsToAddP2 = append(podsToAddP2, podName)
				}
			}
			podsToAdd = append(podsToAdd, podsToAddP2...)

			for _, podName := range podsToAdd {
				if len(podsToKeep) < maxParallelPodLogging {
					process, err := startKubectlProcess(runningPodsMap[podName], previous, &os.ProcAttr{
						Files: []*os.File{inr, outw, outw},
					})
					if err != nil {
						socketWriterError <- err
						return
					}
					processMap[podName] = process
					podsToKeep.Add(podName)
				} else {
					break
				}
			}

			deleteMap := make(map[string]*os.Process, len(podsToDelete))
			for podName := range podsToDelete {
				deleteMap[podName] = processMap[podName]
			}
			deleteProcesses(deleteMap)
			timer.Reset(podCheckInterval)
		}
	}
}

func deleteProcesses(processMap map[string]*os.Process) {
	for podName := range processMap {
		processMap[podName].Signal(os.Interrupt)
	}
	time.Sleep(5 * time.Second)
	for podName := range processMap {
		processMap[podName].Signal(os.Kill)
	}
}

func getCloudWatchLogs(prefix string, verbose bool, socket *websocket.Conn) {
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
	go pumpStdout(socket, socketWriterError, logsReader, verbose, false)

	inr, inw, err := os.Pipe()
	if err != nil {
		errors.Panic(err, "logs", "cloudwatch", "os.pipe")
	}
	defer inr.Close()
	defer inw.Close()

	pumpStdin(socket, inw)
}

func pumpStdin(socket *websocket.Conn, writer io.Writer) {
	socket.SetReadLimit(maxMessageSize)
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

func pumpStdout(socket *websocket.Conn, socketWriterError chan error, reader io.Reader, verbose bool, checkForLastLog bool) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		socket.SetWriteDeadline(time.Now().Add(writeWait))
		logBytes := scanner.Bytes()
		isLastLog := false
		if !verbose {
			logBytes, isLastLog = cleanLogBytes(logBytes)
		}
		if logBytes != nil {
			if !writeSocketBytes(logBytes, socket) {
				break
			}
		}
		if isLastLog && checkForLastLog && !verbose {
			break
		}
	}

	select {
	case err := <-socketWriterError:
		writeSocket(err.Error(), socket)
	default:
	}

	socket.SetWriteDeadline(time.Now().Add(writeWait))
	socket.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	time.Sleep(closeGracePeriod)
	socket.Close()
}

var cortexRegex = regexp.MustCompile(`^\[.*\]?(DEBUG|INFO|WARNING|ERROR|CRITICAL):cortex:`)
var tensorflowRegex = regexp.MustCompile(`^\[.*\]?(DEBUG|INFO|WARNING|ERROR|CRITICAL):tensorflow:`)
var jsonPrefixRegex = regexp.MustCompile(`^\ *?(\{|\[)`)

func formatHeader1(headerString string) *string {
	headerBorder := "\n" + strings.Repeat("-", len(headerString)) + "\n"
	return pointer.String(headerBorder + strings.Title(headerString) + headerBorder)
}

func formatHeader2(headerString string) *string {
	return pointer.String("\n" + strings.ToUpper(string(headerString[0])) + headerString[1:] + "\n")
}

func formatHeader3(headerString string) *string {
	return pointer.String("\n" + strings.ToUpper(string(headerString[0])) + headerString[1:])
}

func extractFromCortexLog(match string, loglevel string, logStr string) (*string, bool) {
	if loglevel == "DEBUG" {
		return nil, false
	}

	cutStr := logStr[len(match):]

	switch cutStr {
	case "Starting":
		return formatHeader3(cutStr), false
	case "Aggregates:":
		return formatHeader2(cutStr), false
	case "Ingesting":
		return formatHeader1(cutStr), false
	case "Aggregating":
		return formatHeader1(cutStr), false
	case "Validating Transformers":
		return formatHeader1(cutStr), false
	case "Generating Training Datasets":
		return formatHeader1(cutStr), false
	case "Caching":
		return formatHeader1(cutStr), false
	case "Training":
		return formatHeader1(cutStr), false
	case "Evaluating":
		return formatHeader1(cutStr), false
	case "Setting up packages":
		return formatHeader1(cutStr), false
	case "Validating packages":
		return formatHeader1(cutStr), false
	case "Caching built packages":
		return formatHeader3(cutStr), false
	}

	matches := jsonPrefixRegex.FindStringSubmatch(cutStr)
	if len(matches) == 2 {
		indent := len(matches[0]) - 1 // matches to curly or square bracket so remove the last char
		indentStr := cutStr[:indent]
		maybeJSON := cutStr[len(indentStr):]
		jsonBytes := []byte(maybeJSON)
		var obytes bytes.Buffer
		err := json.Indent(&obytes, jsonBytes, indentStr, "  ")
		if err == nil {
			fmt.Println("pass json indent")
			ostr := indentStr + string(obytes.String())
			return &ostr, false
		}
	}

	lastLogRe := regexp.MustCompile(`^workload: (\w+), completed: (\S+)`)
	if lastLogRe.MatchString(cutStr) {
		return &cutStr, true
	}

	samplesRe := regexp.MustCompile(`^First (\d+) samples:`)
	if samplesRe.MatchString(cutStr) {
		return formatHeader2(cutStr), false
	}

	if strings.HasPrefix(cutStr, "Transforming ") {
		return formatHeader3(cutStr), false
	}

	if strings.HasPrefix(cutStr, "Reading") {
		return formatHeader3(cutStr), false
	}

	if strings.HasPrefix(cutStr, "Serving model") {
		return formatHeader3(cutStr), false
	}

	if strings.HasPrefix(cutStr, "Prediction failed") {
		return formatHeader2(cutStr), false
	}

	if strings.HasPrefix(cutStr, "Predicting") {
		return formatHeader3(cutStr), false
	}

	if strings.HasPrefix(cutStr, "error:") {
		return pointer.String("\n" + cutStr), false
	}

	if strings.HasPrefix(cutStr, "An error occurred") {
		return formatHeader3(cutStr), false
	}

	return &cutStr, false
}

func extractFromTensorflowLog(match string, loglevel string, logStr string) (*string, bool) {
	if loglevel == "DEBUG" || loglevel == "WARNING" {
		return nil, false
	}

	cutStr := logStr[len(match):]
	if strings.HasPrefix(cutStr, "loss = ") {
		return pointer.String(cutStr), false
	}
	if strings.HasPrefix(cutStr, "Starting evaluation") {
		return formatHeader1("Evaluating"), false
	}
	if strings.HasPrefix(cutStr, "Saving dict for global step") {
		metricsStr := cutStr[strings.Index(cutStr, ":")+1:]
		metricsStr = strings.TrimSpace(metricsStr)
		metrics := strings.Split(metricsStr, ", ")
		outStr := strings.Join(metrics, "\n")
		return pointer.String(outStr), false
	}

	return nil, false
}

func cleanLog(logStr string) (*string, bool) {
	matches := cortexRegex.FindStringSubmatch(logStr)
	if len(matches) == 2 {
		return extractFromCortexLog(matches[0], matches[1], logStr)
	}

	matches = tensorflowRegex.FindStringSubmatch(logStr)
	if len(matches) == 2 {
		return extractFromTensorflowLog(matches[0], matches[1], logStr)
	}

	return nil, false
}

func cleanLogBytes(logBytes []byte) ([]byte, bool) {
	logStr := string(logBytes)
	cleanLogStr, isLastLog := cleanLog(logStr)
	if cleanLogStr == nil {
		return nil, isLastLog
	}
	return []byte(*cleanLogStr), isLastLog
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
