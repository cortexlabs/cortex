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
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/gorilla/websocket"
	"gopkg.in/karalabe/cookiejar.v1/collections/deque"

	awslib "github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/operator/api/context"
	"github.com/cortexlabs/cortex/pkg/operator/api/resource"
	"github.com/cortexlabs/cortex/pkg/operator/config"
)

const (
	socketWriteDeadlineWait = 10 * time.Second
	socketCloseGracePeriod  = 10 * time.Second
	socketMaxMessageSize    = 8192

	maxLogLinesPerRequest = 500
	pollPeriod            = 250 * time.Millisecond
	streamRefreshPeriod   = 2 * time.Second
)

type eventCache struct {
	size       int
	seen       strset.Set
	eventQueue *deque.Deque
}

func NewEventCache(cacheSize int) eventCache {
	return eventCache{
		size:       cacheSize,
		seen:       strset.New(),
		eventQueue: deque.New(),
	}
}

func (c *eventCache) Has(eventID string) bool {
	return c.seen.Has(eventID)
}

func (c *eventCache) PopLeft() {
	eventID := c.eventQueue.PopLeft().(string)
	c.seen.Remove(eventID)
}

func (c *eventCache) Add(eventID string) {
	if c.eventQueue.Size() == c.size {
		c.PopLeft()
	}
	c.seen.Add(eventID)
	c.eventQueue.PushRight(eventID)
}

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

	lastLogTime := time.Now()
	lastLogStreamUpdateTime := time.Now().Add(-1 * streamRefreshPeriod)

	logStreamNames := []string{}
	logStreamSet := strset.New()
	var currentContextID string
	var prefix string
	var err error

	var ctx = CurrentContext(appName)
	eventCache := NewEventCache(10000)

	if ctx == nil {
		writeAndCloseSocket(socket, "\ndeployment "+appName+" not found")
		return
	}

	logGroupName, err := getLogGroupName(ctx, podLabels)
	if err != nil {
		writeAndCloseSocket(socket, err.Error()) // unexpected
		return
	}

	for {
		select {
		case <-podCheckCancel:
			return
		case <-timer.C:
			ctx = CurrentContext(appName)

			if ctx == nil {
				writeAndCloseSocket(socket, "\ndeployment "+appName+" not found")
				continue
			}

			if ctx.ID != currentContextID {
				if len(currentContextID) != 0 {
					if podLabels["workloadType"] == resource.APIType.String() {
						apiName := podLabels["apiName"]
						if _, ok := ctx.APIs[apiName]; !ok {
							writeAndCloseSocket(socket, "\napi "+apiName+" was not found in latest deployment")
							continue
						}
						writeString(socket, "\na new deployment was detected, streaming logs from the latest deployment")
					} else {
						writeAndCloseSocket(socket, "\nlogging non-api workloads is not supported") // unexpected
						continue
					}
				} else {
					lastLogTime, _ = getPodStartTime(podLabels)
				}

				currentContextID = ctx.ID
				writeString(socket, "\nretrieving logs...")
			}

			if lastLogStreamUpdateTime.Add(streamRefreshPeriod).Before(time.Now()) {
				newLogStreamSet, err := getLogStreams(logGroupName)
				if err != nil {
					writeAndCloseSocket(socket, "error encountered while searching for log streams: "+err.Error())
					continue
				}

				if !logStreamSet.IsEqual(newLogStreamSet) {
					lastLogTime = lastLogTime.Add(-streamRefreshPeriod)
					logStreamNames = newLogStreamSet.Slice()
				}

				lastLogStreamUpdateTime = time.Now()
			}

			if len(logStreamNames) == 0 {
				timer.Reset(pollPeriod)
				continue
			}

			endTime := time.Now().Unix() * 1000

			logEventsOutput, err := config.AWS.CloudWatchLogsClient.FilterLogEvents(&cloudwatchlogs.FilterLogEventsInput{
				LogGroupName:   aws.String(logGroupName),
				LogStreamNames: aws.StringSlice(logStreamNames),
				StartTime:      aws.Int64(TimeToMillis(lastLogTime.Add(-pollPeriod))),
				EndTime:        aws.Int64(endTime), // requires milliseconds
				Limit:          aws.Int64(int64(maxLogLinesPerRequest)),
			})

			if err != nil {
				if !awslib.CheckErrCode(err, cloudwatchlogs.ErrCodeResourceNotFoundException) {
					writeAndCloseSocket(socket, "error encountered while fetching logs from cloudwatch: "+err.Error())
					continue
				}
			}

			lastLogTimestampMillis := TimeToMillis(lastLogTime)
			for _, logEvent := range logEventsOutput.Events {
				var log FluentdLog
				json.Unmarshal([]byte(*logEvent.Message), &log)

				if !eventCache.Has(*logEvent.EventId) {
					socket.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("%s %s", *logEvent.LogStreamName, log.Log)))
					if *logEvent.Timestamp > lastLogTimestampMillis {
						lastLogTimestampMillis = *logEvent.Timestamp
					}
					eventCache.Add(*logEvent.EventId)
				}
			}

			lastLogTime = MillisToTime(endTime)
			if len(logEventsOutput.Events) == maxLogLinesPerRequest {
				writeString(socket, "---- Showing at most "+s.Int(maxLogLinesPerRequest)+" lines. Visit AWS cloudwatch logs console and search for \""+prefix+"\" in log group \""+config.Cortex.LogGroup+"\" for complete logs ----")
				lastLogTime = MillisToTime(endTime)
			}

			timer.Reset(pollPeriod)
		}
	}
}

func MillisToTime(epochMillis int64) time.Time {
	return time.Unix(epochMillis/1000, (epochMillis%1000)*1000)
}

func TimeToMillis(t time.Time) int64 {
	return t.UnixNano() / 1000
}

func getLogStreams(logGroupName string) (strset.Set, error) {
	describeLogStreamsOnput, err := config.AWS.CloudWatchLogsClient.DescribeLogStreams(&cloudwatchlogs.DescribeLogStreamsInput{
		Descending:   aws.Bool(true),
		LogGroupName: aws.String(logGroupName),
		OrderBy:      aws.String(cloudwatchlogs.OrderByLastEventTime),
		Limit:        aws.Int64(50),
	})
	if err != nil {
		if !awslib.CheckErrCode(err, cloudwatchlogs.ErrCodeResourceNotFoundException) {
			return nil, err
		}
		return nil, nil
	}

	streams := strset.New()

	for _, stream := range describeLogStreamsOnput.LogStreams {
		streams.Add(*stream.LogStreamName)
	}
	return streams, nil
}

func getPodStartTime(searchLabels map[string]string) (time.Time, error) {
	pods, err := config.Kubernetes.ListPodsByLabels(searchLabels)
	if err != nil {
		return time.Now(), err
	}

	if len(pods) == 0 {
		return time.Now(), nil
	}

	startTime := pods[0].CreationTimestamp.Time
	for _, pod := range pods[1:] {
		if pod.CreationTimestamp.Time.Before(startTime) {
			startTime = pod.CreationTimestamp.Time
		}
	}

	return startTime, nil
}

func getLogGroupName(ctx *context.Context, searchLabels map[string]string) (string, error) {
	if searchLabels["workloadType"] == resource.APIType.String() {
		return ctx.LogGroupName(searchLabels["apiName"]), nil
	}
	return "nil", errors.New("unsupported workload type") // unexpected
}

func writeString(socket *websocket.Conn, message string) {
	socket.WriteMessage(websocket.TextMessage, []byte(message))
}

func closeSocket(socket *websocket.Conn) {
	socket.SetWriteDeadline(time.Now().Add(socketWriteDeadlineWait))
	socket.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	time.Sleep(socketCloseGracePeriod)
}

func writeAndCloseSocket(socket *websocket.Conn, message string) {
	writeString(socket, message)
	closeSocket(socket)
}
