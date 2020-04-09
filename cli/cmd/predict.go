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

package cmd

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/lib/urls"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/spf13/cobra"
)

var _flagPredictDebug bool

var _predictClient = &GenericClient{
	Client: &http.Client{
		Timeout: 600 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	},
}

func init() {
	addEnvFlag(_predictCmd, _generalCommandType)
	_predictCmd.Flags().BoolVar(&_flagPredictDebug, "debug", false, "predict with debug mode")
}

var _predictCmd = &cobra.Command{
	Use:   "predict API_NAME JSON_FILE",
	Short: "make a prediction request using a json file",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		telemetry.Event("cli.predict")

		apiName := args[0]
		jsonPath := args[1]

		httpRes, err := HTTPGet("/get/" + apiName)
		if err != nil {
			exit.Error(err)
		}

		var apiRes schema.GetAPIResponse
		if err = json.Unmarshal(httpRes, &apiRes); err != nil {
			exit.Error(err, "/get"+apiName, string(httpRes))
		}

		totalReady := apiRes.Status.Updated.Ready + apiRes.Status.Stale.Ready
		if totalReady == 0 {
			exit.Error(ErrorAPINotReady(apiName, apiRes.Status.Message()))
		}

		apiEndpoint := urls.Join(apiRes.BaseURL, *apiRes.API.Endpoint)
		if _flagPredictDebug {
			apiEndpoint += "?debug=true"
		}

		predictResponse, err := makePredictRequest(apiEndpoint, jsonPath)
		if err != nil {
			exit.Error(err)
		}

		prettyResp, err := json.Pretty(predictResponse)
		if err != nil {
			exit.Error(err)
		}
		fmt.Println(prettyResp)
	},
}

func makePredictRequest(apiEndpoint string, jsonPath string) (interface{}, error) {
	jsonBytes, err := files.ReadFileBytes(jsonPath)
	if err != nil {
		exit.Error(err)
	}

	payload := bytes.NewBuffer(jsonBytes)
	req, err := http.NewRequest("POST", apiEndpoint, payload)
	if err != nil {
		return nil, errors.Wrap(err, _errStrCantMakeRequest)
	}

	req.Header.Set("Content-Type", "application/json")
	httpResponse, err := _predictClient.MakeRequest(req)
	if err != nil {
		return nil, err
	}

	var predictResponse interface{}
	err = json.DecodeWithNumber(httpResponse, &predictResponse)
	if err != nil {
		return nil, errors.Wrap(err, "prediction response")
	}

	return predictResponse, nil
}
