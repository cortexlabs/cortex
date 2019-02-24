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

	"github.com/cortexlabs/cortex/pkg/api/resource"
	schema "github.com/cortexlabs/cortex/pkg/api/schema"
	s "github.com/cortexlabs/cortex/pkg/api/strings"
	"github.com/cortexlabs/cortex/pkg/operator/aws"
	"github.com/cortexlabs/cortex/pkg/operator/workloads"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
)

func GetAggregate(w http.ResponseWriter, r *http.Request) {
	appName, err := getRequiredQParam("appName", r)
	if RespondIfError(w, err) {
		return
	}
	id, err := getRequiredPathParam("id", r)
	if RespondIfError(w, err) {
		return
	}
	ctx := workloads.CurrentContext(appName)
	if ctx == nil {
		RespondError(w, errors.New(s.ErrAppNotDeployed(appName)))
		return
	}

	aggregate := ctx.Aggregates.OneByID(id)

	if aggregate == nil {
		RespondError(w, errors.New(resource.AggregateType.String(), id, s.ErrNotFound))
		return
	}

	exists, err := aws.IsS3File(aggregate.Key)
	if RespondIfError(w, err, resource.AggregateType.String(), id) {
		return
	}
	if !exists {
		RespondError(w, errors.New(resource.AggregateType.String(), id, s.ErrPending))
		return
	}

	bytes, err := aws.ReadBytesFromS3(aggregate.Key)
	if RespondIfError(w, err, resource.AggregateType.String(), id) {
		return
	}

	response := schema.GetAggregateResponse{Value: bytes}

	Respond(w, response)
}
