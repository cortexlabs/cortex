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
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/json"
	libtime "github.com/cortexlabs/cortex/pkg/lib/time"
	"github.com/cortexlabs/cortex/pkg/lib/zip"
	"github.com/cortexlabs/cortex/pkg/operator/api/schema"
)

var httpTransport = &http.Transport{
	TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
}

var httpClient = &http.Client{
	Timeout:   time.Second * 20,
	Transport: httpTransport,
}

func HTTPGet(endpoint string, qParams ...map[string]string) ([]byte, error) {
	req, err := operatorRequest("GET", endpoint, nil, qParams)
	if err != nil {
		return nil, err
	}
	return makeRequest(req)
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
	return makeRequest(req)
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
		return nil, errors.Wrap(err, errStrCantMakeRequest)
	}

	req, err := operatorRequest("POST", endpoint, body, qParams)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	return makeRequest(req)
}

func addFileToMultipart(fileName string, writer *multipart.Writer, reader io.Reader) error {
	part, err := writer.CreateFormFile(fileName, fileName)
	if err != nil {
		return errors.Wrap(err, errStrCantMakeRequest)
	}

	if _, err = io.Copy(part, reader); err != nil {
		return errors.Wrap(err, errStrCantMakeRequest)
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

func StreamLogs(appName string, resourceName string, resourceType string, verbose bool) error {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	req, err := operatorRequest("GET", "/logs/read", nil, nil)
	if err != nil {
		return err
	}

	values := req.URL.Query()
	values.Set("resourceName", resourceName)
	values.Set("resourceType", resourceType)
	values.Set("appName", appName)
	values.Set("verbose", strconv.FormatBool(verbose))
	req.URL.RawQuery = values.Encode()
	wsURL := req.URL.String()
	wsURL = strings.Replace(wsURL, "http", "ws", 1)

	header := http.Header{}
	header.Set("Authorization", authHeader())
	header.Set("CortexAPIVersion", consts.CortexVersion)

	var dialer = websocket.Dialer{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	connection, response, err := dialer.Dial(wsURL, header)
	if response == nil {
		cliConfig := getValidCliConfig()
		return ErrorFailedToConnect(strings.Replace(cliConfig.CortexURL, "http", "ws", 1))
	}
	defer response.Body.Close()

	if err != nil {
		bodyBytes, err := ioutil.ReadAll(response.Body)
		if err != nil || bodyBytes == nil || string(bodyBytes) == "" {
			cliConfig := getValidCliConfig()
			return ErrorFailedToConnect(strings.Replace(cliConfig.CortexURL, "http", "ws", 1))
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
				os.Exit(1)
			}

			lastLogRe := regexp.MustCompile(`^workload: (\w+), completed: (\S+)`)
			msgStr := string(message)
			if lastLogRe.MatchString(msgStr) {
				match := lastLogRe.FindStringSubmatch(msgStr)
				timestamp, err := time.Parse(time.RFC3339, match[2])
				if err != nil {
					fmt.Println(msgStr)
				} else {
					timestampHuman := libtime.LocalTimestampHuman(&timestamp)
					fmt.Println("\nCompleted on " + timestampHuman)
				}
			} else {
				fmt.Println(msgStr)
			}
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
	cliConfig := getValidCliConfig()
	req, err := http.NewRequest(method, cliConfig.CortexURL+endpoint, body)
	if err != nil {
		return nil, errors.Wrap(err, errStrCantMakeRequest)
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

func makeRequest(request *http.Request) ([]byte, error) {
	request.Header.Set("Authorization", authHeader())
	request.Header.Set("CortexAPIVersion", consts.CortexVersion)

	response, err := httpClient.Do(request)
	if err != nil {
		cliConfig := getValidCliConfig()
		log.Println(err)
		return nil, ErrorFailedToConnect(cliConfig.CortexURL)
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		bodyBytes, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return nil, errors.Wrap(err, errStrRead)
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
		return nil, errors.Wrap(err, errStrRead)
	}
	return bodyBytes, nil
}

func authHeader() string {
	cliConfig := getValidCliConfig()
	return fmt.Sprintf("CortexAWS %s|%s", cliConfig.AWSAccessKeyID, cliConfig.AWSSecretAccessKey)
}
