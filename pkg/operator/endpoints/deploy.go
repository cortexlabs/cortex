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

package endpoints

import (
	"net/http"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/hash"
	"github.com/cortexlabs/cortex/pkg/lib/zip"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types/spec"
)

func Deploy(w http.ResponseWriter, r *http.Request) {
	force := getOptionalBoolQParam("force", false, r)

	configPath := getOptionalQParam("configPath", r)
	if configPath == "" {
		configPath = "api config file"
	}

	configBytes, err := files.ReadReqFile(r, "config")
	if err != nil {
		respondError(w, r, errors.WithStack(err))
		return
	} else if len(configBytes) == 0 {
		respondError(w, r, ErrorFormFileMustBeProvided("config"))
		return
	}

	baseURL, err := operator.APIsBaseURL()
	if err != nil {
		respondError(w, r, err)
		return
	}

	projectBytes, err := files.ReadReqFile(r, "project.zip")
	if err != nil {
		respondError(w, r, err)
		return
	}
	projectID := hash.Bytes(projectBytes)
	projectKey := operator.ProjectKey(projectID)
	projectFileMap, err := zip.UnzipMemToMem(projectBytes)
	if err != nil {
		respondError(w, r, err)
		return
	}

	projectFiles := operator.ClusterProjectFiles{
		ProjectByteMap: projectFileMap,
	}
	apiConfigs, err := spec.ExtractAPIConfigs(configBytes, projectFiles, configPath)
	if err != nil {
		respondError(w, r, err)
		return
	}

	err = operator.ValidateClusterAPIs(apiConfigs, projectFiles)
	if err != nil {
		respondError(w, r, err)
		return
	}

	isProjectUploaded, err := config.AWS.IsS3File(config.Cluster.Bucket, projectKey)
	if err != nil {
		respondError(w, r, err)
		return
	}
	if !isProjectUploaded {
		if err = config.AWS.UploadBytesToS3(projectBytes, config.Cluster.Bucket, projectKey); err != nil {
			respondError(w, r, err)
			return
		}
	}

	results := make([]schema.DeployResult, len(apiConfigs))
	for i, apiConfig := range apiConfigs {
		api, msg, err := operator.UpdateAPI(&apiConfig, projectID, force)
		results[i].Message = msg
		if err != nil {
			results[i].Error = errors.Message(err)
		} else {
			results[i].API = *api
		}
	}

	respond(w, schema.DeployResponse{
		Results: results,
		BaseURL: baseURL,
	})
}
