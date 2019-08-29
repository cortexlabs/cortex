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

package context

import (
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/hash"
	"github.com/cortexlabs/cortex/pkg/lib/zip"
	"github.com/cortexlabs/cortex/pkg/operator/api/context"
	"github.com/cortexlabs/cortex/pkg/operator/api/userconfig"
	"github.com/cortexlabs/cortex/pkg/operator/config"
)

func New(
	userconf *userconfig.Config,
	projectBytes []byte,
	ignoreCache bool,
) (*context.Context, error) {
	files, err := zip.UnzipMemToMem(projectBytes)
	if err != nil {
		return nil, err
	}

	ctx := &context.Context{}
	ctx.CreatedEpoch = time.Now().Unix()

	ctx.CortexConfig = config.Cortex

	ctx.App = getApp(userconf.App)

	deploymentVersion, err := getOrSetDeploymentVersion(ctx.App.Name, ignoreCache)
	if err != nil {
		return nil, err
	}
	ctx.DeploymentVersion = deploymentVersion

	ctx.Root = filepath.Join(
		consts.AppsDir,
		ctx.App.Name,
		consts.DeploymentsDir,
		ctx.DeploymentVersion,
	)

	ctx.MetadataRoot = filepath.Join(
		ctx.Root,
		consts.MetadataDir,
	)

	ctx.StatusPrefix = StatusPrefix(ctx.App.Name)
	pythonPackages, err := loadPythonPackages(files, ctx.DeploymentVersion)
	if err != nil {
		return nil, err
	}
	ctx.PythonPackages = pythonPackages

	apis, err := getAPIs(userconf, ctx.DeploymentVersion, files, pythonPackages)

	if err != nil {
		return nil, err
	}
	ctx.APIs = apis

	err = ctx.Validate()
	if err != nil {
		return nil, err
	}

	ctx.ProjectID = hash.Bytes(projectBytes)
	ctx.ProjectKey = filepath.Join(consts.ProjectDir, ctx.ProjectID, "project.zip")
	if err = config.AWS.UploadBytesToS3(projectBytes, ctx.ProjectKey); err != nil {
		return nil, err
	}
	ctx.ID = calculateID(ctx)
	ctx.Key = ctxKey(ctx.ID, ctx.App.Name)
	return ctx, nil
}

func ctxKey(ctxID string, appName string) string {
	return filepath.Join(
		consts.AppsDir,
		appName,
		consts.ContextsDir,
		ctxID+".msgpack",
	)
}

func calculateID(ctx *context.Context) string {
	ids := []string{}
	ids = append(ids, config.Cortex.ID)
	ids = append(ids, ctx.DeploymentVersion)
	ids = append(ids, ctx.Root)
	ids = append(ids, ctx.StatusPrefix)
	ids = append(ids, ctx.App.ID)
	ids = append(ids, ctx.ProjectID)

	for _, resource := range ctx.AllResources() {
		ids = append(ids, resource.GetID())
	}

	sort.Strings(ids)
	return hash.String(strings.Join(ids, ""))
}

func DownloadContext(ctxID string, appName string) (*context.Context, error) {
	s3Key := ctxKey(ctxID, appName)
	var ctx context.Context

	if err := config.AWS.ReadMsgpackFromS3(&ctx, s3Key); err != nil {
		return nil, err
	}

	return &ctx, nil
}

func StatusPrefix(appName string) string {
	return filepath.Join(
		consts.AppsDir,
		appName,
		consts.ResourceStatusesDir,
	)
}

func StatusKey(resourceID string, workloadID string, appName string) string {
	return filepath.Join(
		StatusPrefix(appName),
		resourceID,
		workloadID,
	)
}

func LatestWorkloadIDKey(resourceID string, appName string) string {
	return filepath.Join(
		StatusPrefix(appName),
		resourceID,
		"latest",
	)
}

func BaseWorkloadKey(workloadID string, appName string) string {
	return filepath.Join(
		consts.AppsDir,
		appName,
		consts.WorkloadSpecsDir,
		workloadID,
	)
}
