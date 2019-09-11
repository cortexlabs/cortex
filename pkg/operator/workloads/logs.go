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
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/gorilla/websocket"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/operator/api/context"
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

func ReadLogs(appName string, apiName string, socket *websocket.Conn) {
	podCheckCancel := make(chan struct{})
	defer close(podCheckCancel)
	go StreamFromCloudWatch(podCheckCancel, appName, apiName, socket)
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

func StreamFromCloudWatch(podCheckCancel chan struct{}, appName string, apiName string, socket *websocket.Conn) {
	timer := time.NewTimer(0)
	defer timer.Stop()
	lastTimestamp := int64(0)

	previousEvents := strset.New()
	wrotePending := false

	var ctx *context.Context
	var latestContextID string
	var podTemplatehash string
	podSearchLabels := map[string]string{
		"appName": appName,
		"apiName": apiName,
	}

	for {
		select {
		case <-podCheckCancel:
			return
		case <-timer.C:
			ctx = CurrentContext(appName)

			if ctx == nil {
				writeString(socket, "\nunable to find running deployments")
				closeSocket(socket)
				return
			}

			if _, ok := ctx.APIs[apiName]; !ok {
				writeString(socket, "\napi "+apiName+" was not found in latest deployment")
				closeSocket(socket)
				return
			}

			if len(latestContextID) != 0 && ctx.ID != latestContextID {
				writeString(socket, "\na new deployment was detected, streaming logs from the latest deployment")
			}

			if ctx.ID != latestContextID {
				latestContextID = ctx.ID
				podSearchLabels["workloadID"] = ctx.APIs[apiName].WorkloadID
				podTemplatehash = ""
				wrotePending = false
			}

			if !wrotePending {
				writeString(socket, "\npending...")
				wrotePending = true
			}

			if len(podTemplatehash) == 0 {
				podTemplatehash, _ = getPodTemplateHash(podSearchLabels)
			}

			if len(podTemplatehash) == 0 {
				timer.Reset(250 * time.Millisecond)
				continue
			}

			prefix := internalAPIName(apiName, appName) + "-" + podTemplatehash

			pods, err := config.Kubernetes.ListPodsByLabels(podSearchLabels)

			streamNames := []string{}
			for i, pod := range pods {
				if i >= 5 {
					return
				}
				containers := pod.Spec.InitContainers
				containers = append(containers, pod.Spec.Containers...)

				for _, container := range containers {
					streamName := fmt.Sprintf("%s_%s", pod.Name, container.Name)
					streamNames = append(streamNames, streamName)
				}
			}

			logEventsOutput, err := config.AWS.CloudWatchLogsClient.FilterLogEvents(&cloudwatchlogs.FilterLogEventsInput{
				LogGroupName:        aws.String(config.Cortex.LogGroup),
				LogStreamNamePrefix: aws.String(prefix),
				StartTime:           aws.Int64(lastTimestamp),
				EndTime:             aws.Int64(time.Now().Unix() * 1000),
				Limit:               aws.Int64(10000),
			})

			if err != nil {
				awsErr := errors.Cause(err).(awserr.Error)
				if awsErr.Code() == "ResourceNotFoundException" {
					if !wrotePending {
						writeString(socket, "pending...")
						wrotePending = true
					}
				} else {
					writeString(socket, "error encountered while fetching logs from cloudwatch")
					closeSocket(socket)
					continue
				}
			}

			newEvents := strset.New()
			for _, logEvent := range logEventsOutput.Events {
				var log FluentdLog
				json.Unmarshal([]byte(*logEvent.Message), &log)

				if !previousEvents.Has(*logEvent.EventId) {
					socket.WriteMessage(websocket.TextMessage, []byte(log.Log[:len(log.Log)]))

					if *logEvent.Timestamp > lastTimestamp {
						lastTimestamp = *logEvent.Timestamp
					}
				}
				newEvents.Add(*logEvent.EventId)
			}

			if len(logEventsOutput.Events) == 10000 {
				socket.WriteMessage(websocket.TextMessage, []byte("---- showing at most 10 000 lines navigate to cloudwatch for the complete logstream ---- "))
			}

			previousEvents = newEvents
			timer.Reset(250 * time.Millisecond)
		}
	}
}

func getPodTemplateHash(labels map[string]string) (string, error) {
	pods, err := config.Kubernetes.ListPodsByLabels(labels)
	if err != nil {
		return "", err
	}

	if len(pods) == 0 {
		return "", nil
	}

	return pods[0].GetLabels()["pod-template-hash"], nil
}

func writeString(socket *websocket.Conn, message string) {
	socket.WriteMessage(websocket.TextMessage, []byte(message))
}

func closeSocket(socket *websocket.Conn) {
	socket.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	time.Sleep(socketCloseGracePeriod)
}
