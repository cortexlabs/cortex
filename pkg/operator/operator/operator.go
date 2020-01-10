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

package operator

import (
	"time"

	"github.com/cortexlabs/cortex/pkg/lib/cron"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/parallel"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/operator/config"
)

func Init() error {
	telemetry.Event("operator.init")

	_, err = UpdateMemoryCapacityConfigMap()
	if err != nil {
		return errors.Wrap(err, "init")
	}

	cron.Run(operatorCron, cronErrHandler("operator"), 5*time.Second)
	cron.Run(telemetryCron, cronErrHandler("telemetry"), 1*time.Hour)

	return nil
}

func deleteAPI(apiName string, keepCache bool) bool {
	wasDeployed := config.Kubernetes.DeploymentExists(apiName)

	parallel.Run(
		func() error {
			return deleteK8sResources(apiName)
		},
		func() error {
			if keepCache {
				return nil
			}
			return deleteS3Resources(apiName)
		},
	)

	return wasDeployed
}
