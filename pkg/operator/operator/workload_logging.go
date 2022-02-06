/*
Copyright 2022 Cortex Labs, Inc.

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
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"text/template"
	"time"

	"github.com/cortexlabs/cortex/pkg/config"
	awslib "github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/operator/lib/routines"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/status"
	"github.com/gorilla/websocket"
)

const (
	_socketWriteDeadlineWait = 10 * time.Second
	_socketCloseGracePeriod  = 10 * time.Second
	_socketMaxMessageSize    = 8192
	_readBufferSize          = 4096

	_pendingPodCheckInterval = 1 * time.Second
	_pollPeriod              = 250 * time.Millisecond
)

func timeString(t time.Time) string {
	return fmt.Sprintf("%sT%02d*3a%02d*3a%02d", t.Format("2006-01-02"), t.Hour(), t.Minute(), t.Second())
}

var _apiLogURLTemplate *template.Template = template.Must(template.New("api_log_url_template").Parse(strings.TrimSpace(`
https://console.{{.Partition}}.com/cloudwatch/home?region={{.Region}}#logsV2:logs-insights$3FqueryDetail$3D$257E$2528end$257E0$257Estart$257E-3600$257EtimeType$257E$2527RELATIVE$257Eunit$257E$2527seconds$257EeditorString$257E$2527fields*20*40timestamp*2c*20message*0a*7c*20filter*20cortex.labels.apiName*3d*22{{.APIName}}*22*0a*7c*20sort*20*40timestamp*20asc*0a$257Esource$257E$2528$257E$2527{{.LogGroup}}$2529$2529
`)))

var _completedJobLogURLTemplate *template.Template = template.Must(template.New("completed_job_log_url_template").Parse(strings.TrimSpace(`
https://console.{{.Partition}}.com/cloudwatch/home?region={{.Region}}#logsV2:logs-insights$3FqueryDetail$3D$257E$2528end$257E$2527{{.EndTime}}$257Estart$257E$2527{{.StartTime}}$257EtimeType$257E$2527ABSOLUTE$257Etz$257E$2527Local$257EeditorString$257E$2527fields*20*40timestamp*2c*20message*0a*7c*20filter*20cortex.labels.apiName*3d*22{{.APIName}}*22*20and*20cortex.labels.jobID*3d*22{{.JobID}}*22*0a*7c*20sort*20*40timestamp*20asc*0a$257Esource$257E$2528$257E$2527{{.LogGroup}}$2529$2529
`)))

var _inProgressJobLogsURLTemplate *template.Template = template.Must(template.New("in_progress_job_log_url_template").Parse(strings.TrimSpace(`
https://console.{{.Partition}}.com/cloudwatch/home?region={{.Region}}#logsV2:logs-insights$3FqueryDetail$3D$257E$2528end$257E0$257Estart$257E-3600$257EtimeType$257E$2527RELATIVE$257Eunit$257E$2527seconds$257EeditorString$257E$2527fields*20*40timestamp*2c*20message*0a*7c*20filter*20cortex.labels.apiName*3d*22{{.APIName}}*22*20and*20cortex.labels.jobID*3d*22{{.JobID}}*22*0a*7c*20sort*20*40timestamp*20asc*0a$257Esource$257E$2528$257E$2527{{.LogGroup}}$2529$2529
`)))

type apiLogURLTemplateArgs struct {
	Partition string
	Region    string
	LogGroup  string
	APIName   string
}

type completedJobLogURLTemplateArgs struct {
	Partition string
	Region    string
	StartTime string
	EndTime   string
	LogGroup  string
	APIName   string
	JobID     string
}

type inProgressJobLogURLTemplateArgs struct {
	Partition string
	Region    string
	LogGroup  string
	APIName   string
	JobID     string
}

func completedBatchJobLogsURL(args completedJobLogURLTemplateArgs) (string, error) {
	buf := &bytes.Buffer{}
	err := _completedJobLogURLTemplate.Execute(buf, args)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(buf.String()), nil
}

func inProgressBatchJobLogsURL(args inProgressJobLogURLTemplateArgs) (string, error) {
	buf := &bytes.Buffer{}
	err := _inProgressJobLogsURLTemplate.Execute(buf, args)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(buf.String()), nil
}

func APILogURL(api spec.API) (string, error) {
	partition := "aws.amazon"
	region := config.ClusterConfig.Region
	if awslib.PartitionFromRegion(region) == "aws-us-gov" {
		partition = "amazonaws-us-gov"
	}
	logGroup := config.ClusterConfig.ClusterName

	args := apiLogURLTemplateArgs{
		Partition: partition,
		Region:    region,
		LogGroup:  logGroup,
		APIName:   api.Name,
	}

	buf := &bytes.Buffer{}
	err := _apiLogURLTemplate.Execute(buf, args)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(buf.String()), nil
}

func BatchJobLogURL(apiName string, jobStatus status.BatchJobStatus) (string, error) {
	partition := "aws.amazon"
	region := config.ClusterConfig.Region
	if awslib.PartitionFromRegion(region) == "aws-us-gov" {
		partition = "amazonaws-us-gov"
	}
	logGroup := config.ClusterConfig.ClusterName

	if jobStatus.EndTime != nil {
		endTime := *jobStatus.EndTime
		endTime = endTime.Add(60 * time.Second)
		return completedBatchJobLogsURL(completedJobLogURLTemplateArgs{
			Partition: partition,
			Region:    region,
			StartTime: timeString(jobStatus.StartTime),
			EndTime:   timeString(endTime),
			LogGroup:  logGroup,
			APIName:   apiName,
			JobID:     jobStatus.ID,
		})
	}
	return inProgressBatchJobLogsURL(inProgressJobLogURLTemplateArgs{
		Partition: partition,
		Region:    region,
		LogGroup:  logGroup,
		APIName:   apiName,
		JobID:     jobStatus.ID,
	})
}

func TaskJobLogURL(apiName string, jobStatus status.TaskJobStatus) (string, error) {
	partition := "aws.amazon"
	region := config.ClusterConfig.Region
	if awslib.PartitionFromRegion(region) == "aws-us-gov" {
		partition = "amazonaws-us-gov"
	}
	logGroup := config.ClusterConfig.ClusterName
	if jobStatus.EndTime != nil {
		endTime := *jobStatus.EndTime
		endTime = endTime.Add(60 * time.Second)
		return completedBatchJobLogsURL(completedJobLogURLTemplateArgs{
			Partition: partition,
			Region:    region,
			StartTime: timeString(jobStatus.StartTime),
			EndTime:   timeString(endTime),
			LogGroup:  logGroup,
			APIName:   apiName,
			JobID:     jobStatus.ID,
		})
	}
	return inProgressBatchJobLogsURL(inProgressJobLogURLTemplateArgs{
		Partition: partition,
		Region:    region,
		LogGroup:  logGroup,
		APIName:   apiName,
		JobID:     jobStatus.ID,
	})
}

func waitForPodToBeNotPending(podName string, cancelListener chan struct{}, socket *websocket.Conn) bool {
	wrotePending := false
	timer := time.NewTimer(0)

	for true {
		select {
		case <-cancelListener:
			return false
		case <-timer.C:
			pod, err := config.K8s.GetPod(podName)
			if err != nil {
				writeAndCloseSocket(socket, fmt.Sprintf("error encountered while attempting to stream logs from pod %s\n%s", podName, err.Error()))
				return false
			}
			if pod == nil {
				writeAndCloseSocket(socket, "unable to find pod")
				return false
			}
			podStatus := k8s.GetPodStatus(pod)
			if podStatus == k8s.PodStatusPending {
				if !wrotePending {
					writeString(socket, "waiting for pod to initialize ...\n")
				}
				wrotePending = true
				timer.Reset(_pendingPodCheckInterval)
				continue
			}
			return true
		}
	}
	return false
}

type jsonMessage struct {
	Message string `json:"message"`
	ExcInfo string `json:"exc_info"`
}

func startKubectlProcess(podName string, cancelListener chan struct{}, socket *websocket.Conn) {
	shouldContinue := waitForPodToBeNotPending(podName, cancelListener, socket)
	if !shouldContinue {
		return
	}

	cmd := exec.Command("/usr/local/bin/kubectl", "-n="+config.K8s.Namespace, "logs", "--all-containers", podName, "--follow")

	cleanup := func() {
		// trigger a wait on the child process and while the process is being waited on,
		// send the kill signal to allow cleanup to happen correctly and prevent zombie processes
		time.AfterFunc(1*time.Second, func() {
			cmd.Process.Kill()
		})

		cmd.Process.Wait()
	}
	defer cleanup()

	logStream, err := cmd.StdoutPipe()
	if err != nil {
		telemetry.Error(errors.ErrorUnexpected(err.Error()))
		operatorLogger.Error(err)
	}

	cmd.Start()

	routines.RunWithPanicHandler(func() {
		pumpStdout(socket, logStream)
	})

	<-cancelListener
}

func pumpStdout(socket *websocket.Conn, reader io.Reader) {
	// it seems like if the buffer is maxed out with no ending token, the scanner just exits.
	// increase the buffer used by the scanner to accommodate larger log lines (a common issue when printing progress)
	p := make([]byte, 1024*1024)
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(p, 1024*1024)
	for scanner.Scan() {
		logBytes := scanner.Bytes()
		var message jsonMessage
		err := json.Unmarshal(logBytes, &message)
		if err != nil {
			writeString(socket, string(logBytes)+"\n")
		} else {
			writeString(socket, message.Message+"\n")
			if message.ExcInfo != "" {
				writeString(socket, message.ExcInfo+"\n")
			}
		}
	}

	closeSocket(socket)
}

func StreamLogsFromRandomPod(podSearchLabels map[string]string, socket *websocket.Conn) {
	pods, err := config.K8s.ListPodsByLabels(podSearchLabels)
	if err != nil {
		writeAndCloseSocket(socket, err.Error())
		return
	}
	if len(pods) == 0 {
		writeAndCloseSocket(socket, "there are currently no pods running for this workload; please visit your logging dashboard for historical logs\n")
		return
	}

	cancelListener := make(chan struct{})
	defer close(cancelListener)
	routines.RunWithPanicHandler(func() {
		startKubectlProcess(pods[0].Name, cancelListener, socket)
	})
	pumpStdin(socket)
	cancelListener <- struct{}{}
}

func pumpStdin(socket *websocket.Conn) {
	socket.SetReadLimit(_socketMaxMessageSize)
	for {
		_, _, err := socket.ReadMessage()
		if err != nil {
			break
		}
	}
}

func writeString(socket *websocket.Conn, message string) {
	socket.WriteMessage(websocket.TextMessage, []byte(message))
}

func writeAndCloseSocket(socket *websocket.Conn, message string) {
	writeString(socket, message)
	closeSocket(socket)
}

func closeSocket(socket *websocket.Conn) {
	socket.SetWriteDeadline(time.Now().Add(_socketWriteDeadlineWait))
	socket.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	time.Sleep(_socketCloseGracePeriod)
}
