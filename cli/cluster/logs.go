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

package cluster

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strings"

	"github.com/cortexlabs/cortex/cli/lib/routines"
	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	"github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types"
	"github.com/gorilla/websocket"
)

func StreamLogs(operatorConfig OperatorConfig, apiName string) error {
	return streamLogs(operatorConfig, "/logs/"+apiName)
}

func StreamJobLogs(operatorConfig OperatorConfig, apiName string, jobID string) error {
	return streamLogs(operatorConfig, "/logs/"+apiName, map[string]string{"jobID": jobID})
}

func streamLogs(operatorConfig OperatorConfig, path string, qParams ...map[string]string) error {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	req, err := operatorRequest(operatorConfig, "GET", path, nil, qParams...)
	if err != nil {
		return err
	}

	values := req.URL.Query()
	if operatorConfig.Telemetry {
		values.Set("clientID", operatorConfig.ClientID)
	}

	req.URL.RawQuery = values.Encode()
	wsURL := req.URL.String()
	wsURL = strings.Replace(wsURL, "http", "ws", 1)

	header := http.Header{}
	header.Set("CortexAPIVersion", consts.CortexVersion)
	if operatorConfig.Provider == types.AWSProviderType {
		awsClient, err := aws.New()
		if err != nil {
			return err
		}

		authHeader, err := awsClient.IdentityRequestAsHeader()
		if err != nil {
			return err
		}
		header.Set(consts.AuthHeader, authHeader)
	}

	var dialer = websocket.Dialer{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	connection, response, err := dialer.Dial(wsURL, header)
	if err != nil && response == nil {
		return ErrorFailedToConnectOperator(err, operatorConfig.EnvName, strings.Replace(operatorConfig.OperatorEndpoint, "http", "ws", 1))
	}
	defer response.Body.Close()

	if err != nil {
		bodyBytes, err := ioutil.ReadAll(response.Body)
		if err != nil || bodyBytes == nil || string(bodyBytes) == "" {
			return ErrorFailedToConnectOperator(err, operatorConfig.EnvName, strings.Replace(operatorConfig.OperatorEndpoint, "http", "ws", 1))
		}
		var output schema.ErrorResponse
		err = json.Unmarshal(bodyBytes, &output)
		if err != nil || output.Message == "" {
			return ErrorOperatorStreamResponseUnknown(string(bodyBytes), response.StatusCode)
		}
		return errors.WithStack(&errors.Error{
			Kind:        output.Kind,
			Message:     output.Message,
			NoTelemetry: true,
		})
	}
	defer connection.Close()

	done := make(chan struct{})
	handleConnection(connection, done)
	closeConnection(connection, done, interrupt)
	return nil
}

func handleConnection(connection *websocket.Conn, done chan struct{}) {
	routines.RunWithPanicHandler(func() {
		defer close(done)
		for {
			_, message, err := connection.ReadMessage()
			if err != nil {
				exit.Error(ErrorOperatorSocketRead(err))
			}
			fmt.Print(string(message))
		}
	}, false)
}

func closeConnection(connection *websocket.Conn, done chan struct{}, interrupt chan os.Signal) {
	for {
		select {
		case <-done:
			return
		case <-interrupt:
			connection.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			return
		}
	}
}
