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
	"fmt"
	"net/http"

	"github.com/cortexlabs/cortex/cli/cluster"
	"github.com/cortexlabs/cortex/cli/local"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/lib/urls"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types"
	"github.com/spf13/cobra"
)

var (
	_flagPredictEnv   string
	_flagPredictDebug bool
)

func predictInit() {
	_predictCmd.Flags().SortFlags = false
	_predictCmd.Flags().StringVarP(&_flagPredictEnv, "env", "e", getDefaultEnv(_generalCommandType), "environment to use")
	_predictCmd.Flags().BoolVar(&_flagPredictDebug, "debug", false, "predict with debug mode")
}

var _predictCmd = &cobra.Command{
	Use:   "predict API_NAME JSON_FILE",
	Short: "make a prediction request using a json file",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		telemetry.Event("cli.predict")
		printEnvIfNotSpecified(_flagPredictEnv)

		apiName := args[0]
		jsonPath := args[1]

		env := MustReadOrConfigureEnv(_flagPredictEnv)
		var apiRes schema.GetAPIResponse
		var err error
		var apiEndpoint string
		if env.Provider == types.AWSProviderType {
			apiRes, err = cluster.GetAPI(MustGetOperatorConfig(env.Name), apiName)
			if err != nil {
				exit.Error(err)
			}
			apiEndpoint = urls.Join(apiRes.BaseURL, *apiRes.API.Endpoint)

		} else {
			apiRes, err = local.GetAPI(apiName)
			if err != nil {
				exit.Error(err)
			}
			apiEndpoint = "http://" + urls.Join(apiRes.BaseURL, "predict")
		}

		totalReady := apiRes.Status.Updated.Ready + apiRes.Status.Stale.Ready
		if totalReady == 0 {
			exit.Error(ErrorAPINotReady(apiName, apiRes.Status.Message()))
		}

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
	httpResponse, err := MakeRequest(req)
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
