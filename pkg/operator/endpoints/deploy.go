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
	"fmt"
	"github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/operator/pb"
	"google.golang.org/grpc/metadata"
	"io"
	"net/http"
	"strings"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/hash"
	"github.com/cortexlabs/cortex/pkg/lib/zip"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types"
	"github.com/cortexlabs/cortex/pkg/types/spec"
)

func deploy(force bool, configPath string, configBytes, projectBytes []byte) (*schema.DeployResponse, error) {
	baseURL, err := operator.APIsBaseURL()
	if err != nil {
		return nil, err
	}
	projectID := hash.Bytes(projectBytes)
	projectKey := spec.ProjectKey(projectID)
	projectFileMap, err := zip.UnzipMemToMem(projectBytes)
	if err != nil {
		return nil, err
	}
	projectFiles := operator.ProjectFiles{
		ProjectByteMap: projectFileMap,
		ConfigFilePath: configPath,
	}

	apiConfigs, err := spec.ExtractAPIConfigs(configBytes, types.AWSProviderType, projectFiles, configPath)
	if err != nil {
		return nil, err
	}

	err = operator.ValidateClusterAPIs(apiConfigs, projectFiles)
	if err != nil {
		return nil, err
	}

	isProjectUploaded, err := config.AWS.IsS3File(config.Cluster.Bucket, projectKey)
	if err != nil {
		return nil, err
	}
	if !isProjectUploaded {
		if err = config.AWS.UploadBytesToS3(projectBytes, config.Cluster.Bucket, projectKey); err != nil {
			return nil, err
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
	return &schema.DeployResponse{
		Results: results,
		BaseURL: baseURL,
	}, nil
}

func Deploy(w http.ResponseWriter, r *http.Request) {
	force := getOptionalBoolQParam("force", false, r)

	configPath, err := getRequiredQueryParam("configPath", r)
	if err != nil {
		respondError(w, r, errors.WithStack(err))
		return
	}

	configBytes, err := files.ReadReqFile(r, "config")
	if err != nil {
		respondError(w, r, errors.WithStack(err))
		return
	} else if len(configBytes) == 0 {
		respondError(w, r, ErrorFormFileMustBeProvided("config"))
		return
	}

	projectBytes, err := files.ReadReqFile(r, "project.zip")
	if err != nil {
		respondError(w, r, err)
		return
	}

	res, err := deploy(force, configPath, configBytes, projectBytes)

	if err != nil {
		respondError(w, r, err)
		return
	}
	respond(w, *res)
}

func (ep *endpoint) Deploy(srv pb.EndPointService_DeployServer) error {
	ctx := srv.Context()
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return fmt.Errorf("empty ctx")
	}
	force := true
	value := md.Get("force")
	if len(value) == 0 {
		force = false
	}
	if strings.ToLower(value[0]) != "true" {
		force = false
	}

	value = md.Get("configPath")
	if len(value) == 0 {
		return errors.WithStack(ErrorQueryParamRequired("configPath"))
	}
	var configBytes, projectBytes []byte
	for {
		select {
			case <- ctx.Done():
				return ctx.Err()
			default:
		}
		req, err := srv.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		configBytes = append(configBytes, req.GetConfig()...)
		projectBytes = append(projectBytes, req.GetProject()...)
	}
	if len(configBytes) == 0 {
		return ErrorFormFileMustBeProvided("config")
	}

	res, err := deploy(force, value[0], configBytes, projectBytes)
	if err != nil {
		return err
	}
	result, _ := json.Marshal(res)
	srv.SendAndClose(&pb.DeployResponse{Response: result})
}
