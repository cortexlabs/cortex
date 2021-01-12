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

package operator

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"time"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/routines"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/gorilla/websocket"
)

const (
	_socketWriteDeadlineWait = 10 * time.Second
	_socketCloseGracePeriod  = 10 * time.Second
	_socketMaxMessageSize    = 8192
	_readBufferSize          = 4096

	_waitForPod = 1 * time.Second
	_pollPeriod = 250 * time.Millisecond
)

// returns whether pod should be logged and if yes, should the logs include previous container logs (only needed if pod fails)
func waitForPodToBeRunning(podName string, podCheckCancel chan struct{}, socket *websocket.Conn) bool {
	wrotePending := false
	timer := time.NewTimer(0)

	for true {
		select {
		case <-podCheckCancel:
			return false
		case <-timer.C:
			pod, err := config.K8s.GetPod(podName)
			if err != nil {
				writeAndCloseSocket(socket, fmt.Sprintf("error encountered while attempting to stream logs from pod %s\n%s", podName, err.Error()))
				return false
			}
			if pod == nil {
				writeAndCloseSocket(socket, fmt.Sprintf("unable to find replica or worker, please try again", podName))
				return false
			}
			podStatus := k8s.GetPodStatus(pod)
			if podStatus == k8s.PodStatusPending {
				if !wrotePending {
					writeString(socket, "waiting for replica/worker to initialize ...\n")
				}
				wrotePending = true
				timer.Reset(_waitForPod)
				continue
			} else if podStatus == k8s.PodStatusRunning || podStatus == k8s.PodStatusInitializing {
				return true
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

func startKubectlProcess(podName string, podCheckCancel chan struct{}, socket *websocket.Conn) {
	shouldContinue := waitForPodToBeRunning(podName, podCheckCancel, socket)
	if !shouldContinue {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	kubectlArgs := []string{"/usr/local/bin/kubectl", "-n=" + "default", "logs", "--all-containers", podName, "--follow"}

	cmd := exec.CommandContext(ctx, kubectlArgs[0], kubectlArgs[1:]...)

	logStream, err := cmd.StdoutPipe()
	if err != nil {
		telemetry.Error(errors.ErrorUnexpected(err.Error()))
		Logger.Error(err)
	}
	cmd.Start()

	go pumpStdout(socket, logStream)

waitForCancel:
	for true {
		select {
		case <-podCheckCancel:
			break waitForCancel
		}
	}

	go func() {
		time.Sleep(1 * time.Second)
		cmd.Process.Kill()
	}()
	cmd.Process.Wait() // trigger a wait on the process and then kill the process to prevent zombie processes

func pumpStdout(socket *websocket.Conn, reader io.Reader) {
	scanner := bufio.NewScanner(reader)
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

// func pumpStdin(socket *websocket.Conn, writer io.Writer) {
// 	for {
// 		_, _, err := socket.ReadMessage()
// 		if err != nil {
// 			fmt.Println(err.Error())
// 			break
// 		}

// 		break
// 	}
// 	fmt.Println("pumpStdin")
// 	writer.Write([]byte{0x03})
// }

func StreamLogsFromRandomPod(podSearchLabels map[string]string, socket *websocket.Conn) {
	pods, err := config.K8s.ListPodsByLabels(podSearchLabels)
	if err != nil {
		writeAndCloseSocket(socket, err.Error())
		return
	}
	if len(pods) == 0 {
		writeAndCloseSocket(socket, "unable to currently running replicas/workers; please visit your logging dashboard for historical logs\n")
		return
	}

	// inr, inw, err := os.Pipe()
	// if err != nil {
	// 	telemetry.Error(errors.ErrorUnexpected(err.Error()))
	// 	Logger.Error(err)
	// }
	// defer inw.Close()
	// defer inr.Close()

	podCheckCancel := make(chan struct{})
	defer close(podCheckCancel)
	routines.RunWithPanicHandler(func() {
		startKubectlProcess(pods[0].Name, podCheckCancel, socket)
	}, false)
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

func writeBytes(socket *websocket.Conn, message []byte) {
	socket.WriteMessage(websocket.TextMessage, message)
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
