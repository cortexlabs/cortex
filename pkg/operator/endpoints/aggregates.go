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
	"github.com/cortexlabs/cortex/pkg/operator/api/resource"
	schema "github.com/cortexlabs/cortex/pkg/operator/api/schema"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/operator/workloads"
)

func GetAggregate(w http.ResponseWriter, r *http.Request) {
	appName, err := getRequiredQueryParam("appName", r)
	if err != nil {
		RespondError(w, err)
		return
	}

	id, err := getRequiredPathParam("id", r)
	if err != nil {
		RespondError(w, err)
		return
	}

	ctx := workloads.CurrentContext(appName)
	if ctx == nil {
		RespondError(w, ErrorAppNotDeployed(appName))
		return
	}

	aggregate := ctx.Aggregates.OneByID(id)

	if aggregate == nil {
		RespondError(w, resource.ErrorNotFound(id, resource.AggregateType))
		return
	}

	exists, err := config.AWS.IsS3File(aggregate.Key)
	if err != nil {
		RespondError(w, err, resource.AggregateType.String(), id)
		return
	}
	if !exists {
		RespondError(w, errors.Wrap(ErrorPending(), resource.AggregateType.String(), id))
		return
	}

	bytes, err := config.AWS.ReadBytesFromS3(aggregate.Key)
	if err != nil {
		RespondError(w, err, resource.AggregateType.String(), id)
		return
	}

	response := schema.GetAggregateResponse{Value: bytes}

	Respond(w, response)
}
