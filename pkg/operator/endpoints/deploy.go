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

package endpoints

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/hash"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/table"
	"github.com/cortexlabs/cortex/pkg/lib/zip"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
)

func Deploy(w http.ResponseWriter, r *http.Request) {
	refresh := getOptionalBoolQParam("refresh", false, r)
	force := getOptionalBoolQParam("force", false, r)

	configPath := getOptionalQParam("configPath", r)
	if configPath == "" {
		configPath = "api config file"
	}

	configBytes, err := files.ReadReqFile(r, "config")
	if err != nil {
		respondError(w, errors.WithStack(err))
		return
	} else if len(configBytes) == 0 {
		respondError(w, ErrorFormFileMustBeProvided("config"))
		return
	}

	baseURL, err := operator.APIsBaseURL()
	if err != nil {
		respondError(w, err)
		return
	}

	projectBytes, err := files.ReadReqFile(r, "project.zip")
	if err != nil {
		respondError(w, err)
		return
	}
	projectID := hash.Bytes(projectBytes)
	projectKey := operator.ProjectKey(projectID)
	projectFileMap, err := zip.UnzipMemToMem(projectBytes)
	if err != nil {
		respondError(w, err)
		return
	}

	apiConfigs, err := operator.ExtractAPIConfigs(configBytes, projectFileMap, configPath)
	if err != nil {
		respondError(w, err)
		return
	}

	isProjectUploaded, err := config.AWS.IsS3File(projectKey)
	if err != nil {
		respondError(w, err)
		return
	}
	if !isProjectUploaded {
		if err = config.AWS.UploadBytesToS3(projectBytes, projectKey); err != nil {
			respondError(w, err)
			return
		}
	}

	// TODO parallelize?
	results := make([]schema.DeployResult, len(apiConfigs))
	for i, apiConfig := range apiConfigs {
		results[i].API, results[i].Message, results[i].Error = operator.UpdateAPI(&apiConfig, projectID, refresh, force)
	}

	respond(w, schema.DeployResponse{
		Results: results,
		BaseURL: baseURL,
		Message: deployMessage(results),
	})
}

func deployMessage(results []schema.DeployResult) string {
	statusMessage := mergeResultMessages(results)

	if didAllResultsError(results) {
		return statusMessage
	}

	apiCommandsMessage := getAPICommandsMessage(results)

	return statusMessage + "\n\n" + apiCommandsMessage
}

func mergeResultMessages(results []schema.DeployResult) string {
	var okMessages []string
	var errMessages []string

	for _, result := range results {
		if result.Error != nil {
			errMessages = append(errMessages, result.Error.Error())
		} else {
			okMessages = append(okMessages, result.Message)
		}
	}

	messages := append(okMessages, errMessages...)

	return strings.Join(messages, "\n")
}

func didAllResultsError(results []schema.DeployResult) bool {
	for _, result := range results {
		if result.Error == nil {
			return false
		}
	}
	return true
}

func getAPICommandsMessage(results []schema.DeployResult) string {
	apiName := "<api_name>"
	if len(results) == 1 {
		apiName = results[0].API.Name
	}

	var items table.KeyValuePairs
	items.Add("cortex get", "(show deployment status)")
	items.Add(fmt.Sprintf("cortex get %s", apiName), "(show api info)")
	items.Add(fmt.Sprintf("cortex logs %s", apiName), "(stream api logs)")

	return items.String(&table.KeyValuePairOpts{
		Delimiter: pointer.String(""),
		NumSpaces: pointer.Int(2),
	})
}
