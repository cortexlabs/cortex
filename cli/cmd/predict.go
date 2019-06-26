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
	libtime "github.com/cortexlabs/cortex/pkg/lib/time"
	"github.com/cortexlabs/cortex/pkg/lib/urls"
	"github.com/cortexlabs/cortex/pkg/operator/api/resource"
)

var predictPrintJSON bool

func init() {
	predictCmd.PersistentFlags().BoolVarP(&predictPrintJSON, "json", "j", false, "print the raw json response")
	addAppNameFlag(predictCmd)
	addEnvFlag(predictCmd)
}

type PredictResponse struct {
	ResourceID  string        `json:"resource_id"`
	Predictions []interface{} `json:"predictions"`
}

var predictCmd = &cobra.Command{
	Use:   "predict API_NAME SAMPLES_FILE",
	Short: "make predictions",
	Long:  "Make predictions.",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		apiName := args[0]
		samplesJSONPath := args[1]

		resourcesRes, err := getResourcesResponse()
		if err != nil {
			errors.Exit(err)
		}

		apiGroupStatus := resourcesRes.APIGroupStatuses[apiName]
		if apiGroupStatus == nil {
			errors.Exit(ErrorAPINotFound(apiName))
		}
		if apiGroupStatus.ActiveStatus == nil {
			errors.Exit(ErrorAPINotReady(apiName, apiGroupStatus.Message()))
		}

		apiPath := apiGroupStatus.ActiveStatus.Path
		apiURL := urls.Join(resourcesRes.APIsBaseURL, apiPath)
		predictResponse, err := makePredictRequest(apiURL, samplesJSONPath)

		if err != nil {
			if strings.Contains(err.Error(), "503 Service Temporarily Unavailable") || strings.Contains(err.Error(), "502 Bad Gateway") {
				errors.Exit(ErrorAPINotReady(apiName, resource.StatusUpdating.Message()))
			}
			errors.Exit(err)
		}

		if predictPrintJSON {
			prettyResp, err := json.Pretty(predictResponse)
			if err != nil {
				errors.Exit(err)
			}

			fmt.Println(prettyResp)
			return
		}

		apiID := predictResponse.ResourceID
		api := resourcesRes.APIStatuses[apiID]

		apiStart := libtime.LocalTimestampHuman(api.Start)
		fmt.Println("\n" + apiName + " was last updated on " + apiStart + "\n")

		if len(predictResponse.Predictions) == 1 {
			fmt.Println("Prediction:")
		} else {
			fmt.Println("Predictions:")
		}

		for _, prediction := range predictResponse.Predictions {
			prettyResp, err := json.Pretty(prediction)
			if err != nil {
				errors.Exit(err)
			}

			fmt.Println(prettyResp)
		}
	},
}

func makePredictRequest(apiURL string, samplesJSONPath string) (*PredictResponse, error) {
	samplesBytes, err := files.ReadFileBytes(samplesJSONPath)
	if err != nil {
		errors.Exit(err)
	}
	payload := bytes.NewBuffer(samplesBytes)
	req, err := http.NewRequest("POST", apiURL, payload)
	if err != nil {
		return nil, errors.Wrap(err, errStrCantMakeRequest)
	}

	req.Header.Set("Content-Type", "application/json")
	httpResponse, err := makeRequest(req)
	if err != nil {
		return nil, err
	}

	var predictResponse PredictResponse
	err = json.DecodeWithNumber(httpResponse, &predictResponse)
	if err != nil {
		return nil, errors.Wrap(err, "prediction response")
	}

	return &predictResponse, nil
}
