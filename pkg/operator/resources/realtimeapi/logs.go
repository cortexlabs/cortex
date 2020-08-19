/*
Copyright 2020 Cortex Labs, Inc.

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

package realtimeapi

import (
	"encoding/json"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	awslib "github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/cache"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	libtime "github.com/cortexlabs/cortex/pkg/lib/time"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/gorilla/websocket"
	kapps "k8s.io/api/apps/v1"
)

const (
	_socketWriteDeadlineWait = 10 * time.Second
	_socketCloseGracePeriod  = 10 * time.Second
	_socketMaxMessageSize    = 8192
	_maxCacheSize            = 10000

	_maxLogLinesPerRequest = 500
	_maxStreamsPerRequest  = 50

	_pollPeriod              = 250 * time.Millisecond
	_logStreamRefreshPeriod  = 10 * time.Second
	_deploymentRefreshPeriod = 30 * time.Second
)

type fluentdLog struct {
	Log string `json:"log"`
}

func ReadLogs(apiName string, socket *websocket.Conn) {
	podCheckCancel := make(chan struct{})
	defer close(podCheckCancel)
	go streamFromCloudWatch(apiName, podCheckCancel, socket)
	pumpStdin(socket)
	podCheckCancel <- struct{}{}
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

func streamFromCloudWatch(apiName string, podCheckCancel chan struct{}, socket *websocket.Conn) {
	logGroupName := getLogGroupName(apiName)
	eventCache := cache.NewFifoCache(_maxCacheSize)
	lastLogStreamRefresh := time.Time{}
	lastDeploymentRefresh := time.Time{}
	lastLogTime := time.Now()
	logStreamNames := strset.New()
	didShowFetchingMessage := false
	didFetchLogs := false
	var deployment *kapps.Deployment

	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-podCheckCancel:
			return
		case <-timer.C:
			if deployment == nil || time.Since(lastDeploymentRefresh) > _deploymentRefreshPeriod {
				var err error
				deployment, err = config.K8s.GetDeployment(operator.K8sName(apiName))
				if err != nil {
					telemetry.Error(err)
					writeAndCloseSocket(socket, "error: "+errors.Message(err))
					continue
				}
				lastDeploymentRefresh = time.Now()
			}

			if deployment == nil {
				writeAndCloseSocket(socket, "\n"+apiName+" not found")
				continue
			}

			if !didShowFetchingMessage {
				writeString(socket, "fetching logs ...")
				didShowFetchingMessage = true
			}

			if time.Since(lastLogStreamRefresh) > _logStreamRefreshPeriod {
				newLogStreamNames, err := getLogStreams(logGroupName)
				if err != nil {
					telemetry.Error(err)
					writeAndCloseSocket(socket, "error encountered while searching for log streams: "+errors.Message(err))
					continue
				}

				if !logStreamNames.IsEqual(newLogStreamNames) {
					lastLogTime = lastLogTime.Add(-_logStreamRefreshPeriod)
					logStreamNames = newLogStreamNames
				}

				lastLogStreamRefresh = time.Now()
			}

			if len(logStreamNames) == 0 {
				timer.Reset(_pollPeriod)
				continue
			}

			if !didFetchLogs {
				lastLogTime = deployment.CreationTimestamp.Time
				didFetchLogs = true
			}

			endTime := libtime.ToMillis(time.Now())

			logEventsOutput, err := config.AWS.CloudWatchLogs().FilterLogEvents(&cloudwatchlogs.FilterLogEventsInput{
				LogGroupName:   aws.String(logGroupName),
				LogStreamNames: aws.StringSlice(logStreamNames.Slice()),
				StartTime:      aws.Int64(libtime.ToMillis(lastLogTime.Add(-_pollPeriod))),
				EndTime:        aws.Int64(endTime),
				Limit:          aws.Int64(int64(_maxLogLinesPerRequest)),
			})

			if err != nil {
				if !awslib.IsErrCode(err, cloudwatchlogs.ErrCodeResourceNotFoundException) {
					telemetry.Error(err)
					writeAndCloseSocket(socket, "error encountered while fetching logs from cloudwatch: "+errors.Message(err))
					continue
				}
			}

			lastLogTimestampMillis := libtime.ToMillis(lastLogTime)
			for _, logEvent := range logEventsOutput.Events {
				var log fluentdLog
				err := json.Unmarshal([]byte(*logEvent.Message), &log)
				if err != nil {
					telemetry.Error(err)
					writeAndCloseSocket(socket, "error encountered while parsing logs from cloudwatch: "+errors.Message(err))
				}

				if !eventCache.Has(*logEvent.EventId) {
					socket.WriteMessage(websocket.TextMessage, []byte(log.Log))
					if *logEvent.Timestamp > lastLogTimestampMillis {
						lastLogTimestampMillis = *logEvent.Timestamp
					}
					eventCache.Add(*logEvent.EventId)
				}
			}

			lastLogTime = libtime.MillisToTime(lastLogTimestampMillis)
			if len(logEventsOutput.Events) == _maxLogLinesPerRequest {
				writeString(socket, "---- Showing at most "+s.Int(_maxLogLinesPerRequest)+" lines. Visit AWS cloudwatch logs console and navigate to log group \""+logGroupName+"\" for complete logs ----")
				lastLogTime = libtime.MillisToTime(endTime)
			}

			timer.Reset(_pollPeriod)
		}
	}
}

func getLogStreams(logGroupName string) (strset.Set, error) {
	describeLogStreamsOutput, err := config.AWS.CloudWatchLogs().DescribeLogStreams(&cloudwatchlogs.DescribeLogStreamsInput{
		OrderBy:      aws.String(cloudwatchlogs.OrderByLastEventTime),
		Descending:   aws.Bool(true),
		LogGroupName: aws.String(logGroupName),
		Limit:        aws.Int64(_maxStreamsPerRequest),
	})
	if err != nil {
		if !awslib.IsErrCode(err, cloudwatchlogs.ErrCodeResourceNotFoundException) {
			return nil, err
		}
		return nil, nil
	}

	streams := strset.New()

	for _, stream := range describeLogStreamsOutput.LogStreams {
		streams.Add(*stream.LogStreamName)
	}
	return streams, nil
}

func getLogGroupName(apiName string) string {
	return config.Cluster.LogGroup + "/" + apiName
}

func writeString(socket *websocket.Conn, message string) {
	socket.WriteMessage(websocket.TextMessage, []byte(message))
}

func closeSocket(socket *websocket.Conn) {
	socket.SetWriteDeadline(time.Now().Add(_socketWriteDeadlineWait))
	socket.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	time.Sleep(_socketCloseGracePeriod)
}

func writeAndCloseSocket(socket *websocket.Conn, message string) {
	writeString(socket, message)
	closeSocket(socket)
}
