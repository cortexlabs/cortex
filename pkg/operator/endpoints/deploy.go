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

	configBytes, err := files.ReadReqFile(r, "config.yaml")
	if err != nil {
		RespondError(w, errors.WithStack(err))
		return
	}

	if len(configBytes) == 0 {
		RespondError(w, ErrorFormFileMustBeProvided("config.yaml"))
		return
	}

	projectBytes, err := files.ReadReqFile(r, "project.zip")
	if err != nil {
		RespondError(w, errors.WithStack(err))
		return
	}

	userconf, err := userconfig.New("cortex.yaml", configBytes, true)
	if err != nil {
		RespondError(w, err)
		return
	}

	ctx, err := ocontext.New(userconf, projectBytes, ignoreCache)
	if err != nil {
		RespondError(w, err)
		return
	}

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

	switch {
	case isUpdating && ignoreCache:
		Respond(w, schema.DeployResponse{Message: ResCachedDeletedDeploymentStarted})
	case isUpdating && !ignoreCache:
		Respond(w, schema.DeployResponse{Message: ResDeploymentUpdated})
	case !isUpdating && ignoreCache:
		Respond(w, schema.DeployResponse{Message: ResCachedDeletedDeploymentStarted})
	case !isUpdating && !ignoreCache && existingCtx == nil:
		Respond(w, schema.DeployResponse{Message: ResDeploymentStarted})
	case !isUpdating && !ignoreCache && existingCtx != nil && !fullCtxMatch:
		Respond(w, schema.DeployResponse{Message: ResDeploymentUpdated})
	case !isUpdating && !ignoreCache && existingCtx != nil && fullCtxMatch:
		Respond(w, schema.DeployResponse{Message: ResDeploymentUpToDate})
	default:
		Respond(w, schema.DeployResponse{Message: ResDeploymentUpdated})
	}
}
