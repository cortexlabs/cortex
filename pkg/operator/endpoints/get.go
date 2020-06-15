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
	"google.golang.org/grpc/metadata"
	"net/http"

	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types/status"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/gorilla/mux"
)

func GetAPIs(w http.ResponseWriter, r *http.Request) {
	res, err := getAPIS()
	if err != nil {
		respondError(w, r, err)
		return
	}
	respond(w, *res)
}

func getAPIS() (*schema.GetAPIsResponse, error) {
	statuses, err := operator.GetAllStatuses()
	if err != nil {
		return nil, err
	}

	apiNames, apiIDs := namesAndIDsFromStatuses(statuses)
	apis, err := operator.DownloadAPISpecs(apiNames, apiIDs)
	if err != nil {
		return nil, err
	}

	allMetrics, err := operator.GetMultipleMetrics(apis)
	if err != nil {
		return nil, err
	}

	baseURL, err := operator.APIsBaseURL()
	if err != nil {
		return nil, err
	}
	return &schema.GetAPIsResponse{
		APIs:       apis,
		Statuses:   statuses,
		AllMetrics: allMetrics,
		BaseURL:    baseURL,
	}, nil
}

func GetAPI(w http.ResponseWriter, r *http.Request) {
	apiName := mux.Vars(r)["apiName"]
	res, err := getAPI(apiName)
	if err != nil {
		respondError(w, r, err)
		return
	}
	respond(w, *res)
}

func getAPI(apiName string) (*schema.GetAPIResponse, error) {
	status, err := operator.GetStatus(apiName)
	if err != nil {
		return nil, err
	}

	api, err := operator.DownloadAPISpec(status.APIName, status.APIID)
	if err != nil {
		return nil, err
	}

	metrics, err := operator.GetMetrics(api)
	if err != nil {
		return nil, err
	}

	baseURL, err := operator.APIsBaseURL()
	if err != nil {
		return nil, err
	}
	return &schema.GetAPIResponse{
		API:          *api,
		Status:       *status,
		Metrics:      *metrics,
		BaseURL:      baseURL,
		DashboardURL: operator.DashboardURL(),
	}, nil
}

func namesAndIDsFromStatuses(statuses []status.Status) ([]string, []string) {
	apiNames := make([]string, len(statuses))
	apiIDs := make([]string, len(statuses))

	for i, status := range statuses {
		apiNames[i] = status.APIName
		apiIDs[i] = status.APIID
	}

	return apiNames, apiIDs
}

func (ep *endpoint) GetAPIs(ctx context.Context, empty *empty.Empty) (*pb.GetAPIsResponse, error) {
	res, err := getAPIS()
	if err != nil {
		return nil, err
	}
	result, _ := json.Marshal(res)
	return &pb.GetAPIsResponse{Response: result},nil
}

func (ep *endpoint) GetAPI(ctx context.Context, empty *empty.Empty) (*pb.GetAPIResponse, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, fmt.Errorf("empty ctx")
	}
	api := md.Get("apiName")
	if len(api) == 0 {
		return nil, fmt.Errorf("empty value apiName")
	}
	res, err := getAPI(api[0])
	if err != nil {
		return nil, err
	}
	result, _ := json.Marshal(res)
	return &pb.GetAPIResponse{Response: result}, nil
}
