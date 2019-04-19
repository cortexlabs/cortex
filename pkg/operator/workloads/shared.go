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

package workloads

import (
	"github.com/cortexlabs/cortex/pkg/operator/api/context"
	"github.com/cortexlabs/cortex/pkg/operator/api/resource"
	"github.com/cortexlabs/cortex/pkg/lib/random"
)

func generateWorkloadID() string {
	// k8s needs all characters to be lower case, and the first to be a letter
	return random.LowercaseLetters(1) + random.LowercaseString(19)
}

func checkResourceCached(res context.ComputedResource, ctx *context.Context) (bool, error) {
	workloadID := res.GetWorkloadID()
	if workloadID == "" {
		return false, nil
	}

	savedStatus, err := getDataSavedStatus(res.GetID(), workloadID, ctx.App.Name)
	if err != nil {
		return false, err
	}
	if savedStatus != nil && savedStatus.ExitCode == resource.ExitCodeDataSucceeded {
		return true, nil
	}
	return false, nil
}
