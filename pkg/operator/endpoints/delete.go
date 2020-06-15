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

package endpoints

import (
	"context"
	"fmt"
	"github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/operator/pb"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc/metadata"
	"net/http"
	"strings"

	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/gorilla/mux"
)

func (ep *endpoint) Delete(ctx context.Context, empty *empty.Empty) (*pb.DeleteResponse, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, fmt.Errorf("empty ctx")
	}
	api := md.Get("apiName")
	if len(api) == 0 {
		return nil, fmt.Errorf("empty value apiName")
	}
	keepCache := md.Get("keepCache")
	cache := true
	if len(keepCache) == 0 {
		cache = false
	}
	if strings.ToLower(keepCache[0]) != "true" {
		cache = false
	}
	response, err := delete(api[0], cache)

	if err != nil {
		// TODO: handle error
		return nil, err
	}
	res, _ := json.Marshal(response)
	return &pb.DeleteResponse{Response: res}, nil
}

func Delete(w http.ResponseWriter, r *http.Request) {
	apiName := mux.Vars(r)["apiName"]
	keepCache := getOptionalBoolQParam("keepCache", false, r)

	response, err := delete(apiName, keepCache)
	if err != nil {
		if fmt.Sprintf("%d", http.StatusNotFound) == err.Error() {
			respondErrorCode(w, r, http.StatusNotFound, operator.ErrorAPINotDeployed(apiName))
			return
		}
		respondError(w, r, err)
		return
	}
	respond(w, *response)
}

func delete(api string, keepCache bool) (*schema.DeleteResponse, error) {
	isDeployed, err := operator.IsAPIDeployed(api)
	if err != nil {
		return nil, err
	}

	if !isDeployed {
		// Delete anyways just to be sure everything is deleted
		go func() {
			err = operator.DeleteAPI(api, keepCache)
			if err != nil {
				telemetry.Error(err)
			}
		}()
		return nil, fmt.Errorf("%d", http.StatusNotFound)
	}

	err = operator.DeleteAPI(api, keepCache)
	if err != nil {
		return nil, err
	}

	return &schema.DeleteResponse{
		Message: fmt.Sprintf("deleting %s", api),
	}, nil
}
