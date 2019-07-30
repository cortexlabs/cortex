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
	"time"

	"github.com/cortexlabs/cortex/pkg/operator/api/schema"
	"github.com/cortexlabs/cortex/pkg/operator/workloads"
)

func GetDeployments(w http.ResponseWriter, r *http.Request) {
	currentContexts := workloads.CurrentContexts()
	deployments := make([]schema.Deployment, len(currentContexts))
	for i, ctx := range currentContexts {
		deployments[i].Name = ctx.App.Name
		status, _ := workloads.GetDeploymentStatus(ctx.App.Name)
		deployments[i].Status = status
		deployments[i].LastUpdated = time.Unix(ctx.CreatedEpoch, 0)
	}

	response := schema.GetDeploymentsResponse{
		Deployments: deployments,
	}

	Respond(w, response)
}
