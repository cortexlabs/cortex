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

package cmd

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/lib/zip"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/gorilla/websocket"
)

type OperatorClient struct {
	*http.Client
}

type GenericClient struct {
	*http.Client
}

var _operatorClient = &OperatorClient{
	Client: &http.Client{
		Timeout: 600 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	},
}

var _apiClient = &GenericClient{
	Client: &http.Client{
		Timeout: 600 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	},
}

func HTTPGet(endpoint string, qParams ...map[string]string) ([]byte, error) {
	req, err := operatorRequest("GET", endpoint, nil, qParams)
	if err != nil {
		return nil, err
	}
	return _operatorClient.MakeRequest(req)
}

func HTTPPostJSONData(endpoint string, requestData interface{}, qParams ...map[string]string) ([]byte, error) {
	jsonRequestData, err := json.Marshal(requestData)
	if err != nil {
		return nil, err
	}
	return HTTPPostJSON(endpoint, jsonRequestData, qParams...)
}

func HTTPPostJSON(endpoint string, jsonRequestData []byte, qParams ...map[string]string) ([]byte, error) {
	payload := bytes.NewBuffer(jsonRequestData)
	req, err := operatorRequest("POST", endpoint, payload, qParams)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return _operatorClient.MakeRequest(req)
}

type HTTPUploadInput struct {
	FilePaths map[string]string
	Bytes     map[string][]byte
}

func HTTPUpload(endpoint string, input *HTTPUploadInput, qParams ...map[string]string) ([]byte, error) {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	for fileName, filePath := range input.FilePaths {
		file, err := files.Open(filePath)
		if err != nil {
			return nil, err
		}

		defer file.Close()
		if err := addFileToMultipart(fileName, writer, file); err != nil {
			return nil, err
		}
	}

	for fileName, fileBytes := range input.Bytes {
		if err := addFileToMultipart(fileName, writer, bytes.NewReader(fileBytes)); err != nil {
			return nil, err
		}
	}

	if err := writer.Close(); err != nil {
		return nil, errors.Wrap(err, _errStrCantMakeRequest)
	}

	req, err := operatorRequest("POST", endpoint, body, qParams)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	return _operatorClient.MakeRequest(req)
}

func addFileToMultipart(fileName string, writer *multipart.Writer, reader io.Reader) error {
	part, err := writer.CreateFormFile(fileName, fileName)
	if err != nil {
		return errors.Wrap(err, _errStrCantMakeRequest)
	}

	if _, err = io.Copy(part, reader); err != nil {
		return errors.Wrap(err, _errStrCantMakeRequest)
	}
	return nil
}

func HTTPUploadZip(endpoint string, zipInput *zip.Input, fileName string, qParams ...map[string]string) ([]byte, error) {
	zipBytes, err := zip.ToMem(zipInput)
	if err != nil {
		return nil, errors.Wrap(err, "failed to zip configuration file")
	}

	uploadInput := &HTTPUploadInput{
		Bytes: map[string][]byte{
			fileName: zipBytes,
		},
	}
	return HTTPUpload(endpoint, uploadInput, qParams...)
}

func StreamLogs(apiName string) error {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	req, err := operatorRequest("GET", "/logs/read", nil, nil)
	if err != nil {
		return err
	}

	values := req.URL.Query()
	values.Set("apiName", apiName)
	if isTelemetryEnabled() {
		values.Set("clientID", clientID())
	}

	req.URL.RawQuery = values.Encode()
	wsURL := req.URL.String()
	wsURL = strings.Replace(wsURL, "http", "ws", 1)

	authHeader, err := authHeader()
	if err != nil {
		return err
	}

	header := http.Header{}
	header.Set("Authorization", authHeader)
	header.Set("CortexAPIVersion", consts.CortexVersion)

	var dialer = websocket.Dialer{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	connection, response, err := dialer.Dial(wsURL, header)
	if err != nil && response == nil {
		return ErrorFailedToConnectOperator(err, strings.Replace(operatorEndpointOrBlank(), "http", "ws", 1))
	}
	defer response.Body.Close()

	if err != nil {
		bodyBytes, err := ioutil.ReadAll(response.Body)
		if err != nil || bodyBytes == nil || string(bodyBytes) == "" {
			return ErrorFailedToConnectOperator(err, strings.Replace(operatorEndpointOrBlank(), "http", "ws", 1))
		}
		var output schema.ErrorResponse
		err = json.Unmarshal(bodyBytes, &output)
		if err != nil || output.Error == "" {
			return errors.New(string(bodyBytes))
		}
		return errors.New(output.Error)
	}
	defer connection.Close()

	done := make(chan struct{})
	handleConnection(connection, done)
	closeConnection(connection, done, interrupt)
	return nil
}

func handleConnection(connection *websocket.Conn, done chan struct{}) {
	go func() {
		defer close(done)
		for {
			_, message, err := connection.ReadMessage()
			if err != nil {
				exit.ErrorNoPrint(err)
			}
			fmt.Println(string(message))
		}
	}()
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

func operatorRequest(method string, endpoint string, body io.Reader, qParams []map[string]string) (*http.Request, error) {
	cliEnvConfig, err := readOrConfigureCLIEnv(_flagEnv)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, cliEnvConfig.OperatorEndpoint+endpoint, body)
	if err != nil {
		return nil, errors.Wrap(err, _errStrCantMakeRequest)
	}

	values := req.URL.Query()
	for _, paramMap := range qParams {
		for key, value := range paramMap {
			values.Set(key, value)
		}
	}
	req.URL.RawQuery = values.Encode()

	return req, nil
}

func (client *OperatorClient) MakeRequest(request *http.Request) ([]byte, error) {
	if isTelemetryEnabled() {
		values := request.URL.Query()
		values.Set("clientID", clientID())
		request.URL.RawQuery = values.Encode()
	}

	authHeader, err := authHeader()
	if err != nil {
		return nil, err
	}

	request.Header.Set("Authorization", authHeader)
	request.Header.Set("CortexAPIVersion", consts.CortexVersion)

	response, err := client.Do(request)
	if err != nil {
		return nil, ErrorFailedToConnectOperator(err, operatorEndpointOrBlank())
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		bodyBytes, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return nil, errors.Wrap(err, _errStrRead)
		}

		var output schema.ErrorResponse
		err = json.Unmarshal(bodyBytes, &output)
		if err != nil || output.Error == "" {
			return nil, errors.New(strings.TrimSpace(string(bodyBytes)))
		}

		return nil, errors.New(output.Error)
	}

	bodyBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, errors.Wrap(err, _errStrRead)
	}
	return bodyBytes, nil
}

func (client *GenericClient) MakeRequest(request *http.Request) ([]byte, error) {
	response, err := client.Do(request)
	if err != nil {
		return nil, errors.Wrap(err, errStrFailedToConnect(*request.URL))
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		bodyBytes, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return nil, errors.Wrap(err, _errStrRead)
		}
		return nil, errors.New(strings.TrimSpace(string(bodyBytes)))
	}

	bodyBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, errors.Wrap(err, _errStrRead)
	}
	return bodyBytes, nil
}

func authHeader() (string, error) {
	cliEnvConfig, err := readOrConfigureCLIEnv(_flagEnv)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("CortexAWS %s|%s", cliEnvConfig.AWSAccessKeyID, cliEnvConfig.AWSSecretAccessKey), err
}

// Returns empty string if not able to get operator endpoint
func operatorEndpointOrBlank() string {
	cliEnvConfig, _ := readCLIEnvConfig(_flagEnv)
	if cliEnvConfig != nil {
		return cliEnvConfig.OperatorEndpoint
	}
	return ""
}
