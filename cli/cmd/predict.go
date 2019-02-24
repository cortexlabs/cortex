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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cortexlabs/cortex/pkg/api/resource"
	s "github.com/cortexlabs/cortex/pkg/api/strings"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	libstring "github.com/cortexlabs/cortex/pkg/lib/strings"
	libtime "github.com/cortexlabs/cortex/pkg/lib/time"
	"github.com/cortexlabs/cortex/pkg/lib/urls"
)

func init() {
	addAppNameFlag(predictCmd)
	addEnvFlag(predictCmd)
}

type PredictResponse struct {
	ResourceID                string                     `json:"resource_id"`
	ClassificationPredictions []ClassificationPrediction `json:"classification_predictions"`
	RegressionPredictions     []RegressionPrediction     `json:"regression_predictions"`
}

type ClassificationPrediction struct {
	PredictedClass         int         `json:"predicted_class"`
	PredictedClassReversed interface{} `json:"predicted_class_reversed"`
	Probabilities          []float64   `json:"probabilities"`
}

type RegressionPrediction struct {
	PredictedValue         float64     `json:"predicted_value"`
	PredictedValueReversed interface{} `json:"predicted_value_reversed"`
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
			errors.Exit(s.ErrAPINotFound(apiName))
		}
		if apiGroupStatus.ActiveStatus == nil {
			errors.Exit(s.ErrAPINotReady(apiName, apiGroupStatus.Message()))
		}

		apiPath := apiGroupStatus.ActiveStatus.Path
		apiURL := urls.URLJoin(resourcesRes.APIsBaseURL, apiPath)
		predictResponse, err := makePredictRequest(apiURL, samplesJSONPath)
		if err != nil {
			if strings.Contains(err.Error(), "503 Service Temporarily Unavailable") || strings.Contains(err.Error(), "502 Bad Gateway") {
				errors.Exit(s.ErrAPINotReady(apiName, resource.StatusAPIUpdating.Message()))
			}
			errors.Exit(err)
		}

		apiID := predictResponse.ResourceID
		api := resourcesRes.APIStatuses[apiID]

		apiStart := libtime.LocalTimestampHuman(api.Start)
		fmt.Println("\n" + apiName + " was last updated on " + apiStart + "\n")

		if predictResponse.ClassificationPredictions != nil {
			if len(predictResponse.ClassificationPredictions) == 1 {
				fmt.Println("Predicted class:")
			} else {
				fmt.Println("Predicted classes:")
			}
			for _, prediction := range predictResponse.ClassificationPredictions {
				if prediction.PredictedClassReversed != nil {
					json, _ := json.Marshal(prediction.PredictedClassReversed)
					fmt.Println(s.TrimPrefixAndSuffix(string(json), "\""))
				} else {
					fmt.Println(prediction.PredictedClass)
				}
			}
		}
		if predictResponse.RegressionPredictions != nil {
			if len(predictResponse.RegressionPredictions) == 1 {
				fmt.Println("Predicted value:")
			} else {
				fmt.Println("Predicted values:")
			}
			for _, prediction := range predictResponse.RegressionPredictions {
				if prediction.PredictedValueReversed != nil {
					json, _ := json.Marshal(prediction.PredictedValueReversed)
					fmt.Println(s.TrimPrefixAndSuffix(string(json), "\""))
				} else {
					fmt.Println(libstring.Round(prediction.PredictedValue, 2, true))
				}
			}
		}
	},
}

func makePredictRequest(apiURL string, samplesJSONPath string) (*PredictResponse, error) {
	samplesBytes, err := ioutil.ReadFile(samplesJSONPath)
	if err != nil {
		errors.Exit(err, s.ErrReadFile(samplesJSONPath))
	}
	payload := bytes.NewBuffer(samplesBytes)
	req, err := http.NewRequest("POST", apiURL, payload)
	if err != nil {
		return nil, errors.Wrap(err, s.ErrCantMakeRequest)
	}

	req.Header.Set("Content-Type", "application/json")
	httpResponse, err := makeRequest(req)
	if err != nil {
		return nil, err
	}

	var predictResponse PredictResponse
	err = json.Unmarshal(httpResponse, &predictResponse)
	if err != nil {
		return nil, errors.Wrap(err, "prediction response", s.ErrUnmarshalJSON)
	}

	return &predictResponse, nil
}
