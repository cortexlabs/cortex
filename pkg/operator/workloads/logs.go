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
	"encoding/json"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/gorilla/websocket"

	awslib "github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/operator/api/context"
	"github.com/cortexlabs/cortex/pkg/operator/config"
)

const (
	socketWriteDeadlineWait = 10 * time.Second
	socketCloseGracePeriod  = 10 * time.Second
	socketMaxMessageSize    = 8192

	maxLogLinesPerRequest = 500
	pollPeriod            = 250 * time.Millisecond
)

func ReadLogs(appName string, podLabels map[string]string, socket *websocket.Conn) {
	podCheckCancel := make(chan struct{})
	defer close(podCheckCancel)
	go StreamFromCloudWatch(podCheckCancel, appName, podLabels, socket)
	pumpStdin(socket)
	podCheckCancel <- struct{}{}
}

func pumpStdin(socket *websocket.Conn) {
	socket.SetReadLimit(socketMaxMessageSize)
	for {
		_, _, err := socket.ReadMessage()
		if err != nil {
			break
		}
	}
}

type FluentdLog struct {
	Log string `json:"log"`
}

func StreamFromCloudWatch(podCheckCancel chan struct{}, appName string, podLabels map[string]string, socket *websocket.Conn) {
	timer := time.NewTimer(0)
	defer timer.Stop()

	lastTimestamp := int64(0)
	previousEvents := strset.New()
	wrotePending := false

	var currentContextID string
	var prefix string
	var ctx *context.Context
	var err error

	for {
		select {
		case <-podCheckCancel:
			return
		case <-timer.C:
			ctx = CurrentContext(appName)

			if ctx == nil {
				writeString(socket, "\ndeployment "+appName+" not found")
				closeSocket(socket)
				continue
			}

			if ctx.ID != currentContextID {
				if len(currentContextID) != 0 {
					if apiName, ok := podLabels["apiName"]; ok {
						if _, ok := ctx.APIs[apiName]; !ok {
							writeString(socket, "\napi "+apiName+" was not found in latest deployment")
							closeSocket(socket)
							continue
						}
						podLabels["workloadID"] = ctx.APIs[apiName].WorkloadID
						writeString(socket, "\na new deployment was detected, streaming logs from the latest deployment")
					} else {
						writeString(socket, "\nnew deployment detected, shutting down log stream") // unexpected for now, should only occur when logging non api resources
						closeSocket(socket)
						continue
					}
				}

				currentContextID = ctx.ID

				pods, err := config.Kubernetes.ListPodsByLabels(podLabels)
				if err != nil {
					writeString(socket, err.Error())
					closeSocket(socket)
					continue
				}

				allPodsPending := true
				for _, pod := range pods {
					if k8s.GetPodStatus(&pod) != k8s.PodStatusPending {
						allPodsPending = false
						break
					}
				}

				if allPodsPending {
					writeString(socket, "\npending...")
					wrotePending = true
				} else {
					wrotePending = false
				}

				prefix = ""
			}

			if len(prefix) == 0 {
				prefix, err = getPrefix(podLabels)
				if err != nil {
					writeString(socket, err.Error())
					closeSocket(socket)
					continue
				}
			}

			if len(prefix) == 0 {
				timer.Reset(pollPeriod)
				continue
			}

			logEventsOutput, err := config.AWS.CloudWatchLogsClient.FilterLogEvents(&cloudwatchlogs.FilterLogEventsInput{
				LogGroupName:        aws.String(config.Cortex.LogGroup),
				LogStreamNamePrefix: aws.String(prefix),
				StartTime:           aws.Int64(lastTimestamp),
				EndTime:             aws.Int64(time.Now().Unix() * 1000), // requires milliseconds
				Limit:               aws.Int64(int64(maxLogLinesPerRequest)),
			})

			if err != nil {
				if awslib.CheckErrCode(err, "ResourceNotFoundException") {
					if !wrotePending {
						writeString(socket, "pending...")
						wrotePending = true
					}
				} else {
					writeString(socket, "error encountered while fetching logs from cloudwatch: "+err.Error())
					closeSocket(socket)
					continue
				}
			}

			newEvents := strset.New()
			for _, logEvent := range logEventsOutput.Events {
				var log FluentdLog
				json.Unmarshal([]byte(*logEvent.Message), &log)

				if !previousEvents.Has(*logEvent.EventId) {
					socket.WriteMessage(websocket.TextMessage, []byte(log.Log))

					if *logEvent.Timestamp > lastTimestamp {
						lastTimestamp = *logEvent.Timestamp
					}
				}
				newEvents.Add(*logEvent.EventId)
			}

			if len(logEventsOutput.Events) == maxLogLinesPerRequest {
				socket.WriteMessage(websocket.TextMessage, []byte("---- Showing at most "+s.Int(maxLogLinesPerRequest)+" lines. Visit AWS cloudwatch logs console and search for \""+prefix+"\" in log group \""+config.Cortex.LogGroup+"\" for complete logs ----"))
			}

			previousEvents = newEvents
			timer.Reset(pollPeriod)
		}
	}
}

func getPrefix(searchLabels map[string]string) (string, error) {
	pods, err := config.Kubernetes.ListPodsByLabels(searchLabels)
	if err != nil {
		return "", err
	}

	if len(pods) == 0 {
		return "", nil
	}

	podLabels := pods[0].GetLabels()
	if apiName, ok := podLabels["apiName"]; ok {
		if podTemplateHash, ok := podLabels["pod-template-hash"]; ok {
			return internalAPIName(apiName, podLabels["appName"]) + "-" + podTemplateHash, nil
		}
		return "", nil // unexpected, pod template hash not set yet
	}
	return pods[0].Name, nil // unexpected, logging non api resources
}

func writeString(socket *websocket.Conn, message string) {
	socket.SetWriteDeadline(time.Now().Add(socketWriteDeadlineWait))
	socket.WriteMessage(websocket.TextMessage, []byte(message))
}

func closeSocket(socket *websocket.Conn) {
	socket.SetWriteDeadline(time.Now().Add(socketWriteDeadlineWait))
	socket.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	time.Sleep(socketCloseGracePeriod)
}
