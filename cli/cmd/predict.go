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
	"fmt"
	"net/http"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/lib/urls"
)

var predictDebug bool

func init() {
	addAppNameFlag(predictCmd)
	addEnvFlag(predictCmd)
	predictCmd.Flags().BoolVar(&predictDebug, "debug", false, "predict with debug mode")
}

var predictCmd = &cobra.Command{
	Use:   "predict API_NAME SAMPLE_FILE",
	Short: "make a prediction request using a json file",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		telemetry.ReportEvent("cli.predict", nil)

		apiName := args[0]
		sampleJSONPath := args[1]

		appName, err := AppNameFromFlagOrConfig()
		if err != nil {
			telemetry.ExitErr(err)
		}

		resourcesRes, err := getResourcesResponse(appName)
		if err != nil {
			telemetry.ExitErr(err)
		}

		apiGroupStatus := resourcesRes.APIGroupStatuses[apiName]

		// Check for prefix match
		if apiGroupStatus == nil {
			var matchedName string
			for name := range resourcesRes.APIGroupStatuses {
				if strings.HasPrefix(name, apiName) {
					if matchedName != "" {
						telemetry.ExitErr(ErrorAPINotFound(apiName)) // duplicates
					}
					matchedName = name
				}
			}

			if matchedName == "" {
				telemetry.ExitErr(ErrorAPINotFound(apiName))
			}

			if resourcesRes.Context.APIs[matchedName] == nil {
				telemetry.ExitErr(ErrorAPINotFound(apiName))
			}

			apiGroupStatus = resourcesRes.APIGroupStatuses[matchedName]
			apiName = matchedName
		}

		api := resourcesRes.Context.APIs[apiName]
		if api == nil {
			telemetry.ExitErr(ErrorAPINotFound(apiName))
		}

		if apiGroupStatus.ActiveStatus == nil {
			telemetry.ExitErr(ErrorAPINotReady(apiName, apiGroupStatus.Message()))
		}

		apiURL := urls.Join(resourcesRes.APIsBaseURL, *api.Endpoint)
		if predictDebug {
			apiURL += "?debug=true"
		}
		predictResponse, err := makePredictRequest(apiURL, sampleJSONPath)
		if err != nil {
			if strings.Contains(err.Error(), "503 Service Temporarily Unavailable") || strings.Contains(err.Error(), "502 Bad Gateway") {
				telemetry.ExitErr(ErrorAPINotReady(apiName, "creating"))
			}
			telemetry.ExitErr(err)
		}

		prettyResp, err := json.Pretty(predictResponse)
		if err != nil {
			telemetry.ExitErr(err)
		}
		fmt.Println(prettyResp)
	},
}

func makePredictRequest(apiURL string, sampleJSONPath string) (interface{}, error) {
	sampleBytes, err := files.ReadFileBytes(sampleJSONPath)
	if err != nil {
		telemetry.ExitErr(err)
	}
	payload := bytes.NewBuffer(sampleBytes)
	req, err := http.NewRequest("POST", apiURL, payload)
	if err != nil {
		return nil, errors.Wrap(err, errStrCantMakeRequest)
	}

	req.Header.Set("Content-Type", "application/json")
	httpResponse, err := httpClient.makeRequest(req)
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
