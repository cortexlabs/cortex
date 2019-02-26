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

	"github.com/cortexlabs/cortex/pkg/api/context"
	"github.com/cortexlabs/cortex/pkg/api/schema"
	s "github.com/cortexlabs/cortex/pkg/api/strings"
	"github.com/cortexlabs/cortex/pkg/api/userconfig"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/zip"
	"github.com/cortexlabs/cortex/pkg/operator/argo"
	"github.com/cortexlabs/cortex/pkg/operator/aws"
	ocontext "github.com/cortexlabs/cortex/pkg/operator/context"
	"github.com/cortexlabs/cortex/pkg/operator/workloads"
)

func Deploy(w http.ResponseWriter, r *http.Request) {
	ignoreCache := getOptionalBoolQParam("ignoreCache", false, r)
	force := getOptionalBoolQParam("force", false, r)

	ctx, err := getContext(r, ignoreCache)
	if RespondIfError(w, err) {
		return
	}

	newWf, err := workloads.Create(ctx)
	if RespondIfError(w, err) {
		return
	}

	existingWf, err := workloads.GetWorkflow(ctx.App.Name)
	if RespondIfError(w, err) {
		return
	}
	isRunning := false
	if existingWf != nil {
		isRunning = argo.IsRunning(existingWf)
	}

	if isRunning {
		if newWf.Labels["ctxID"] == existingWf.Labels["ctxID"] {
			prevCtx := workloads.CurrentContext(ctx.App.Name)
			if context.APIResourcesAndComputesMatch(ctx, prevCtx) {
				respondDeploy(w, s.ResDeploymentRunning)
				return
			}
		}
		if !force {
			respondDeploy(w, s.ResDifferentDeploymentRunning)
			return
		}
	}

	err = aws.UploadMsgpackToS3(ctx.ToSerial(), ctx.Key)
	if RespondIfError(w, err, ctx.App.Name, "upload context") {
		return
	}

	err = workloads.Run(newWf, ctx, existingWf)
	if RespondIfError(w, err) {
		return
	}

	switch {
	case isRunning && ignoreCache:
		respondDeploy(w, s.ResDeploymentStoppedCacheDeletedDeploymentStarted)
	case isRunning && !ignoreCache && argo.NumTasks(newWf) == 0:
		respondDeploy(w, s.ResDeploymentStoppedDeploymentUpToDate)
	case isRunning && !ignoreCache && argo.NumTasks(newWf) != 0:
		respondDeploy(w, s.ResDeploymentStoppedDeploymentStarted)
	case !isRunning && ignoreCache:
		respondDeploy(w, s.ResCachedDeletedDeploymentStarted)
	case !isRunning && !ignoreCache && argo.NumTasks(newWf) == 0:
		if existingWf != nil && existingWf.Labels["ctxID"] == newWf.Labels["ctxID"] {
			respondDeploy(w, s.ResDeploymentUpToDate)
			return
		}
		respondDeploy(w, s.ResDeploymentUpdated)
	case !isRunning && !ignoreCache && argo.NumTasks(newWf) != 0:
		respondDeploy(w, s.ResDeploymentStarted)
	}
}

func respondDeploy(w http.ResponseWriter, message string) {
	response := schema.DeployResponse{Message: message}
	Respond(w, response)
}

func getContext(r *http.Request, ignoreCache bool) (*context.Context, error) {
	envName, err := getRequiredQParam("environment", r)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	zipBytes, err := files.ReadReqFile(r, "config.zip")
	if err != nil {
		return nil, errors.Wrap(err)
	}
	if len(zipBytes) == 0 {
		return nil, errors.New(s.ErrFormFileMustBeProvided("config.zip"))
	}

	zipContents, err := zip.UnzipMemToMem(zipBytes)
	if err != nil {
		return nil, errors.Wrap(err, "form file", "config.zip")
	}

	config, err := userconfig.New(zipContents, envName)
	if err != nil {
		return nil, err
	}

	ctx, err := ocontext.New(config, zipContents, ignoreCache)
	if err != nil {
		return nil, err
	}

	return ctx, nil
}
