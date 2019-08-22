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
	"time"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	libtime "github.com/cortexlabs/cortex/pkg/lib/time"
	"github.com/cortexlabs/cortex/pkg/operator/config"
)

func getOrSetDeploymentVersion(appName string, ignoreCache bool) (string, error) {
	deploymentVersionFileKey := filepath.Join(
		consts.AppsDir,
		appName,
		"deployment_version",
	)

	if ignoreCache {
		deploymentVersion := libtime.Timestamp(time.Now())
		err := config.AWS.UploadStringToS3(deploymentVersion, deploymentVersionFileKey)
		if err != nil {
			return "", errors.Wrap(err, "deployment version") // unexpected error
		}
		return deploymentVersion, nil
	}

	deploymentVersion, err := config.AWS.ReadStringFromS3(deploymentVersionFileKey)
	if err != nil {
		if !aws.IsNoSuchKeyErr(err) {
			return "", errors.Wrap(err, "deployment version") // unexpected error
		}
		deploymentVersion = libtime.Timestamp(time.Now())
		err := config.AWS.UploadStringToS3(deploymentVersion, deploymentVersionFileKey)
		if err != nil {
			return "", errors.Wrap(err, "deployment version") // unexpected error
		}
	}
	return deploymentVersion, nil
}
