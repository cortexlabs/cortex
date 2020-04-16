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

package cluster

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/lib/zip"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
)

type OperatorClient struct {
	*http.Client
}

type GenericClient struct {
	*http.Client
}

type OperatorConfig struct {
	Telemetry          bool
	ClientID           string
	EnvName            string
	OperatorEndpoint   string
	AWSAccessKeyID     string
	AWSSecretAccessKey string
}

func (oc OperatorConfig) AuthHeader() string {
	return fmt.Sprintf("CortexAWS %s|%s", oc.AWSAccessKeyID, oc.AWSSecretAccessKey)
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

func HTTPGet(operatorConfig OperatorConfig, endpoint string, qParams ...map[string]string) ([]byte, error) {
	req, err := operatorRequest(operatorConfig, "GET", endpoint, nil, qParams)
	if err != nil {
		return nil, err
	}
	return _operatorClient.MakeRequest(operatorConfig, req)
}

func HTTPPostObjAsJSON(operatorConfig OperatorConfig, endpoint string, requestData interface{}, qParams ...map[string]string) ([]byte, error) {
	jsonRequestData, err := json.Marshal(requestData)
	if err != nil {
		return nil, err
	}
	return HTTPPostJSON(operatorConfig, endpoint, jsonRequestData, qParams...)
}

func HTTPPostJSON(operatorConfig OperatorConfig, endpoint string, jsonRequestData []byte, qParams ...map[string]string) ([]byte, error) {
	payload := bytes.NewBuffer(jsonRequestData)
	req, err := operatorRequest(operatorConfig, "POST", endpoint, payload, qParams)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return _operatorClient.MakeRequest(operatorConfig, req)
}

func HTTPPostNoBody(operatorConfig OperatorConfig, endpoint string, qParams ...map[string]string) ([]byte, error) {
	req, err := operatorRequest(operatorConfig, "POST", endpoint, nil, qParams)
	if err != nil {
		return nil, err
	}
	return _operatorClient.MakeRequest(operatorConfig, req)
}

func HTTPDelete(operatorConfig OperatorConfig, endpoint string, qParams ...map[string]string) ([]byte, error) {
	req, err := operatorRequest(operatorConfig, "DELETE", endpoint, nil, qParams)
	if err != nil {
		return nil, err
	}
	return _operatorClient.MakeRequest(operatorConfig, req)
}

type HTTPUploadInput struct {
	FilePaths map[string]string
	Bytes     map[string][]byte
}

func HTTPUpload(operatorConfig OperatorConfig, endpoint string, input *HTTPUploadInput, qParams ...map[string]string) ([]byte, error) {
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

	req, err := operatorRequest(operatorConfig, "POST", endpoint, body, qParams)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	return _operatorClient.MakeRequest(operatorConfig, req)
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

func HTTPUploadZip(operatorConfig OperatorConfig, endpoint string, zipInput *zip.Input, fileName string, qParams ...map[string]string) ([]byte, error) {
	zipBytes, err := zip.ToMem(zipInput)
	if err != nil {
		return nil, errors.Wrap(err, "failed to zip configuration file")
	}

	uploadInput := &HTTPUploadInput{
		Bytes: map[string][]byte{
			fileName: zipBytes,
		},
	}
	return HTTPUpload(operatorConfig, endpoint, uploadInput, qParams...)
}

func operatorRequest(operatorConfig OperatorConfig, method string, endpoint string, body io.Reader, qParams []map[string]string) (*http.Request, error) {
	req, err := http.NewRequest(method, operatorConfig.OperatorEndpoint+endpoint, body)
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

func (client *OperatorClient) MakeRequest(operatorConfig OperatorConfig, request *http.Request) ([]byte, error) {
	if operatorConfig.Telemetry {
		values := request.URL.Query()
		values.Set("clientID", operatorConfig.ClientID)
		request.URL.RawQuery = values.Encode()
	}

	request.Header.Set("Authorization", operatorConfig.AuthHeader())
	request.Header.Set("CortexAPIVersion", consts.CortexVersion)

	response, err := client.Do(request)
	if err != nil {
		return nil, ErrorFailedToConnectOperator(err, operatorConfig.EnvName, operatorConfig.OperatorEndpoint)
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		bodyBytes, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return nil, errors.Wrap(err, _errStrRead)
		}

		var output schema.ErrorResponse
		err = json.Unmarshal(bodyBytes, &output)
		if err != nil || output.Message == "" {
			return nil, ErrorOperatorResponseUnknown(string(bodyBytes), response.StatusCode)
		}

		return nil, errors.WithStack(&errors.Error{
			Kind:        output.Kind,
			Message:     output.Message,
			NoTelemetry: true,
		})
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
		return nil, ErrorResponseUnknown(string(bodyBytes), response.StatusCode)
	}

	bodyBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, errors.Wrap(err, _errStrRead)
	}
	return bodyBytes, nil
}
