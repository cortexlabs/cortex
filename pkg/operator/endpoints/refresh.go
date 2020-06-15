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
	"fmt"
	"github.com/cortexlabs/cortex/pkg/lib/json"
	"google.golang.org/grpc/metadata"
	"net/http"
	"context"
	"strings"

	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/operator/pb"
	"github.com/golang/protobuf/ptypes/empty"

	"github.com/gorilla/mux"
)

func Refresh(w http.ResponseWriter, r *http.Request) {
	apiName := mux.Vars(r)["apiName"]
	force := getOptionalBoolQParam("force", false, r)

	msg, err := operator.RefreshAPI(apiName, force)
	if err != nil {
		respondError(w, r, err)
		return
	}

	response := schema.RefreshResponse{
		Message: msg,
	}
	respond(w, response)
}

func (ep *endpoint) Refresh(ctx context.Context, empty *empty.Empty) (*pb.RefreshResponse, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, fmt.Errorf("empty ctx")
	}
	apiName := md.Get("apiName")
	if len(apiName) == 0 {
		return nil, fmt.Errorf("empty value apiName")
	}
	keep := md.Get("force")
	force := true
	if len(keep) == 0 {
		force = false
	}
	if strings.ToLower(keep[0]) != "true" {
		force = false
	}
	msg, err := operator.RefreshAPI(apiName, force)
	if err != nil {
		return nil, err
	}

	response := &schema.RefreshResponse{
		Message: msg,
	}
	res, _ := json.Marshal(response)
	return &pb.RefreshResponse{Response: res}, nil
}