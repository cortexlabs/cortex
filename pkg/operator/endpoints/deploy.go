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
	"net/http"
	"strings"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/operator/api/context"
	"github.com/cortexlabs/cortex/pkg/operator/api/resource"
	"github.com/cortexlabs/cortex/pkg/operator/api/schema"
	"github.com/cortexlabs/cortex/pkg/operator/api/userconfig"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	ocontext "github.com/cortexlabs/cortex/pkg/operator/context"
	"github.com/cortexlabs/cortex/pkg/operator/workloads"
)

func Deploy(w http.ResponseWriter, r *http.Request) {
	config.Telemetry.ReportEvent("endpoint.deploy")

	ignoreCache := getOptionalBoolQParam("ignoreCache", false, r)
	force := getOptionalBoolQParam("force", false, r)

	configBytes, err := files.ReadReqFile(r, "cortex.yaml")
	if err != nil {
		RespondError(w, errors.WithStack(err))
		return
	}

	if len(configBytes) == 0 {
		RespondError(w, ErrorFormFileMustBeProvided("cortex.yaml"))
		return
	}

	projectBytes, err := files.ReadReqFile(r, "project.zip")

	userconf, err := userconfig.New("cortex.yaml", configBytes)
	if err != nil {
		RespondError(w, err)
		return
	}

	err = userconf.Validate(projectBytes)
	if err != nil {
		RespondError(w, err)
		return
	}

	ctx, err := ocontext.New(userconf, projectBytes, ignoreCache)
	if err != nil {
		RespondError(w, err)
		return
	}

	err = workloads.PopulateWorkloadIDs(ctx)
	if err != nil {
		RespondError(w, err)
		return
	}

	existingCtx := workloads.CurrentContext(ctx.App.Name)

	fullCtxMatch := false
	if existingCtx != nil && existingCtx.ID == ctx.ID && context.APIResourcesAndComputesMatch(ctx, existingCtx) {
		fullCtxMatch = true
	}

	err = workloads.ValidateDeploy(ctx)
	if err != nil {
		RespondError(w, err)
		return
	}

	deploymentStatus, err := workloads.GetDeploymentStatus(ctx.App.Name)
	if err != nil {
		RespondError(w, err)
		return
	}

	isUpdating := deploymentStatus == resource.UpdatingDeploymentStatus

	if isUpdating {
		if fullCtxMatch {
			Respond(w, schema.DeployResponse{Message: ResDeploymentUpToDateUpdating})
			return
		}
		if !force {
			Respond(w, schema.DeployResponse{Message: ResDifferentDeploymentUpdating})
			return
		}
	}

	err = config.AWS.UploadMsgpackToS3(ctx, ctx.Key)
	if err != nil {
		RespondError(w, err, ctx.App.Name, "upload context")
		return
	}

	err = workloads.Run(ctx)
	if err != nil {
		RespondError(w, err)
		return
	}

	apisBaseURL, err := workloads.APIsBaseURL()
	if err != nil {
		RespondError(w, err)
		return
	}

	deployResponse := schema.DeployResponse{Context: ctx, APIsBaseURL: apisBaseURL}
	switch {
	case isUpdating && ignoreCache:
		deployResponse.Message = ResCachedDeletedDeploymentStarted
	case isUpdating && !ignoreCache:
		deployResponse.Message = ResDeploymentUpdated
	case !isUpdating && ignoreCache:
		deployResponse.Message = ResCachedDeletedDeploymentStarted
	case !isUpdating && !ignoreCache && existingCtx == nil:
		deployResponse.Message = ResDeploymentStarted
	case !isUpdating && !ignoreCache && existingCtx != nil && !fullCtxMatch:
		deployResponse.Message = ResDeploymentUpdated
	case !isUpdating && !ignoreCache && existingCtx != nil && fullCtxMatch:
		deployResponse.Message = ResDeploymentUpToDate
	default:
		deployResponse.Message = ResDeploymentUpdated
	}

	deployResponse.OperationsMessage = operationsMessage(existingCtx, ctx)

	Respond(w, deployResponse)
}

func operationsMessage(existingCtx *context.Context, currentCtx *context.Context) string {
	existingEndpoints := make(map[string]string, len(existingCtx.APIs)) // endpoint -> API ID + API compute ID
	for _, api := range existingCtx.APIs {
		existingEndpoints[*api.Endpoint] = api.ID + api.Compute.ID()
	}

	currentEndpoints := make(map[string]string, len(currentCtx.APIs)) // endpoint -> API ID + API compute ID
	for _, api := range currentCtx.APIs {
		currentEndpoints[*api.Endpoint] = api.ID + api.Compute.ID()
	}

	newEndpoints := strset.New()
	updatedEndpoints := strset.New()
	deletedEndpoints := strset.New()
	unchangedEndpoints := strset.New()

	for endpoint, currentID := range currentEndpoints {
		if existingID, ok := existingEndpoints[endpoint]; ok {
			if currentID == existingID {
				unchangedEndpoints.Add(endpoint)
			} else {
				updatedEndpoints.Add(endpoint)
			}
		} else {
			newEndpoints.Add(endpoint)
		}
	}

	for endpoint := range existingEndpoints {
		if _, ok := currentEndpoints[endpoint]; ok {
			continue
		}
		deletedEndpoints.Add(endpoint)
	}

	var strs []string
	for endpoint := range newEndpoints {
		strs = append(strs, "created endpoint: "+endpoint)
	}
	for endpoint := range updatedEndpoints {
		strs = append(strs, "updated endpoint: "+endpoint)
	}
	for endpoint := range deletedEndpoints {
		strs = append(strs, "deleted endpoint: "+endpoint)
	}
	for endpoint := range unchangedEndpoints {
		strs = append(strs, "endpoint not changed: "+endpoint)
	}

	return strings.Join(strs, "\n")
}
