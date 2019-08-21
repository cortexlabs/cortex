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

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/zip"
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

	ctx, err := getContext(r, ignoreCache)
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

	deploymentStatus, err := workloads.GetDeploymentStatus(ctx.App.Name)
	if err != nil {
		RespondError(w, err)
		return
	}

	isUpdating := deploymentStatus == resource.UpdatingDeploymentStatus

	if isUpdating {
		if fullCtxMatch {
			respondDeploy(w, ResDeploymentUpToDateUpdating)
			return
		}
		if !force {
			respondDeploy(w, ResDifferentDeploymentUpdating)
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

	switch {
	case isUpdating && ignoreCache:
		respondDeploy(w, ResCachedDeletedDeploymentStarted)
	case isUpdating && !ignoreCache:
		respondDeploy(w, ResDeploymentUpdated)
	case !isUpdating && ignoreCache:
		respondDeploy(w, ResCachedDeletedDeploymentStarted)
	case !isUpdating && !ignoreCache && existingCtx == nil:
		respondDeploy(w, ResDeploymentStarted)
	case !isUpdating && !ignoreCache && existingCtx != nil && !fullCtxMatch:
		respondDeploy(w, ResDeploymentUpdated)
	case !isUpdating && !ignoreCache && existingCtx != nil && fullCtxMatch:
		respondDeploy(w, ResDeploymentUpToDate)
	default:
		respondDeploy(w, ResDeploymentUpdated) // unexpected
	}
}

func respondDeploy(w http.ResponseWriter, message string) {
	response := schema.DeployResponse{Message: message}
	Respond(w, response)
}

func getContext(r *http.Request, ignoreCache bool) (*context.Context, error) {
	zipBytes, err := files.ReadReqFile(r, "config.zip")
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if len(zipBytes) == 0 {
		return nil, ErrorFormFileMustBeProvided("config.zip")
	}

	zipContents, err := zip.UnzipMemToMem(zipBytes)
	if err != nil {
		return nil, errors.Wrap(err, "form file", "config.zip")
	}

	config, err := userconfig.New(zipContents)
	if err != nil {
		return nil, err
	}

	ctx, err := ocontext.New(config, zipContents, ignoreCache)
	if err != nil {
		return nil, err
	}

	return ctx, nil
}
