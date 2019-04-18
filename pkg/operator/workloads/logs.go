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
	"regexp"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	corev1 "k8s.io/api/core/v1"

	libaws "github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	cc "github.com/cortexlabs/cortex/pkg/operator/cortexconfig"
	"github.com/cortexlabs/cortex/pkg/operator/k8s"
	"github.com/cortexlabs/cortex/pkg/operator/telemetry"
)

const (
	writeWait        = 10 * time.Second
	closeGracePeriod = 10 * time.Second
	maxMessageSize   = 8192
)

func ReadLogs(appName string, workloadID string, verbose bool, socket *websocket.Conn) {
	wrotePending := false

	for true {
		allPods, err := k8s.ListPodsByLabels(map[string]string{
			"appName":    appName,
			"workloadID": workloadID,
			"userFacing": "true",
		})
		if err != nil {
			writeSocket(err.Error(), socket)
			return
		}

		// Prefer running API pods if any exist
		var pods []corev1.Pod
		for _, pod := range allPods {
			if pod.Labels["workloadType"] == WorkloadTypeAPI && k8s.GetPodStatus(&pod) != k8s.PodStatusRunning {
				continue
			}
			pods = append(pods, pod)
		}
		if len(pods) == 0 {
			pods = allPods
		}

		if len(pods) == 1 {
			getKubectlLogs(&pods[0], verbose, wrotePending, socket)
			return
		}

		if len(pods) > 1 {
			if !writeSocket(fmt.Sprintf("%d pods available, streaming logs for one of them:", len(pods)), socket) {
				return
			}
			for _, pod := range pods {
				if pod.Status.Phase != "Pending" {
					getKubectlLogs(&pod, verbose, wrotePending, socket)
					return
				}
			}
			getKubectlLogs(&pods[0], verbose, wrotePending, socket)
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
			getKubectlLogs(failedArgoPod, true, false, socket)
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

func getKubectlLogs(pod *corev1.Pod, verbose bool, wrotePending bool, socket *websocket.Conn) {
	cmdPath := "/usr/local/bin/kubectl"

	if pod.Status.Phase == "Pending" {
		if !wrotePending {
			if !writeSocket("\nPending", socket) {
				return
			}
		}
		k8s.WaitForPodRunning(pod.Name, 1)
	}

	var args []string
	if pod.Labels["workloadType"] == WorkloadTypeAPI && pod.Labels["userFacing"] == "true" {
		args = []string{"kubectl", "-n=" + cc.Namespace, "logs", "--follow=true", pod.Name, apiContainerName}
	} else {
		args = []string{"kubectl", "-n=" + cc.Namespace, "logs", "--follow=true", pod.Name}
	}

	outr, outw, err := os.Pipe()
	if err != nil {
		errors.Panic(err, "logs", "kubectl", "os.pipe")
	}
	defer outr.Close()
	defer outw.Close()

	inr, inw, err := os.Pipe()
	if err != nil {
		errors.Panic(err, "logs", "kubectl", "os.pipe")
	}
	defer inr.Close()
	defer inw.Close()

	process, err := os.StartProcess(cmdPath, args, &os.ProcAttr{
		Files: []*os.File{inr, outw, outw},
	})
	if err != nil {
		errors.Panic(err, strings.Join(args, " "))
	}

	go pumpStdout(socket, outr, verbose, true)
	pumpStdin(socket, inw)
	stopProcess(process)
}

func getCloudWatchLogs(prefix string, verbose bool, socket *websocket.Conn) {
	logs, err := libaws.Client.GetLogs(prefix)
	if err != nil {
		telemetry.ReportError(err)
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
	go pumpStdout(socket, logsReader, verbose, false)

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

func pumpStdout(socket *websocket.Conn, reader io.Reader, verbose bool, checkForLastLog bool) {
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

	socket.SetWriteDeadline(time.Now().Add(writeWait))
	socket.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	time.Sleep(closeGracePeriod)
	socket.Close()
}

var cortexRegex = regexp.MustCompile(`^?(DEBUG|INFO|WARNING|ERROR|CRITICAL):cortex:`)
var tensorflowRegex = regexp.MustCompile(`^?(DEBUG|INFO|WARNING|ERROR|CRITICAL):tensorflow:`)

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
