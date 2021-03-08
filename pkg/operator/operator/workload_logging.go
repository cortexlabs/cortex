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

package operator

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"time"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/operator/lib/routines"
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
				writeAndCloseSocket(socket, "unable to find replica/worker")
				return false
			}
			podStatus := k8s.GetPodStatus(pod)
			if podStatus == k8s.PodStatusPending {
				if !wrotePending {
					writeString(socket, "waiting for replica/worker to initialize ...\n")
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
		writeAndCloseSocket(socket, "there are currently no replicas/workers running for this api or job; please visit your logging dashboard for historical logs\n")
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
