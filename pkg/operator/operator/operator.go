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

package operator

import (
	"time"

	"github.com/cortexlabs/cortex/pkg/lib/cron"
	ocron "github.com/cortexlabs/cortex/pkg/operator/cron"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/operator/resources/syncapi"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
)

func Init() error {
	telemetry.Event("operator.init")

	_, err := updateMemoryCapacityConfigMap()
	if err != nil {
		return errors.Wrap(err, "init")
	}

	deployments, err := config.K8s.ListDeploymentsWithLabelKeys("apiName")
	if err != nil {
		return err
	}

	for _, deployment := range deployments {
		if userconfig.KindFromString(deployment.Labels["apiKind"]) == userconfig.SyncAPIKind {
			if err := syncapi.UpdateAutoscalerCron(&deployment); err != nil {
				return err
			}
		}
	}

	cron.Run(deleteEvictedPods, ocron.ErrorHandler("delete evicted pods"), 12*time.Hour)
	cron.Run(operatorTelemetry, ocron.ErrorHandler("operator telemetry"), 1*time.Hour)

	return nil
}
