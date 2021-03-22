/*
Copyright 2021 Cortex Labs, Inc.

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

package main

import (
	"net/http"
	"time"

	"github.com/cortexlabs/cortex/pkg/lib/cron"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/operator/endpoints"
	"github.com/cortexlabs/cortex/pkg/operator/lib/exit"
	"github.com/cortexlabs/cortex/pkg/operator/lib/logging"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/operator/resources/asyncapi"
	"github.com/cortexlabs/cortex/pkg/operator/resources/job/batchapi"
	"github.com/cortexlabs/cortex/pkg/operator/resources/job/taskapi"
	"github.com/cortexlabs/cortex/pkg/operator/resources/realtimeapi"
	"github.com/cortexlabs/cortex/pkg/types"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var operatorLogger = logging.GetOperatorLogger()

const _operatorPortStr = "8888"

func main() {
	if err := config.Init(); err != nil {
		exit.ErrorNoTelemetry(errors.Wrap(err, "init"))
	}

	telemetry.Event("operator.init", map[string]interface{}{"provider": config.Provider})

	cron.Run(operator.DeleteEvictedPods, operator.ErrorHandler("delete evicted pods"), time.Hour)

	switch config.Provider {
	case types.AWSProviderType:
		cron.Run(operator.InstanceTelemetryAWS, operator.ErrorHandler("instance telemetry"), 1*time.Hour)
	case types.GCPProviderType:
		cron.Run(operator.InstanceTelemetryGCP, operator.ErrorHandler("instance telemetry"), 1*time.Hour)
	}

	if config.Provider == types.AWSProviderType {
		if config.IsManaged() {
			_, err := operator.UpdateMemoryCapacityConfigMap()
			if err != nil {
				exit.Error(errors.Wrap(err, "init"))
			}
		}

		cron.Run(batchapi.ManageJobResources, operator.ErrorHandler("manage batch jobs"), batchapi.ManageJobResourcesCronPeriod)
	}
	cron.Run(taskapi.ManageJobResources, operator.ErrorHandler("manage task jobs"), taskapi.ManageJobResourcesCronPeriod)

	deployments, err := config.K8s.ListDeploymentsWithLabelKeys("apiName")
	if err != nil {
		exit.Error(errors.Wrap(err, "init"))
	}

	for i := range deployments {
		deployment := deployments[i]
		apiKind := deployment.Labels["apiKind"]
		if userconfig.KindFromString(apiKind) == userconfig.RealtimeAPIKind ||
			userconfig.KindFromString(apiKind) == userconfig.AsyncAPIKind {
			apiID := deployment.Labels["apiID"]
			apiName := deployment.Labels["apiName"]
			api, err := operator.DownloadAPISpec(apiName, apiID)
			if err != nil {
				exit.Error(errors.Wrap(err, "init"))
			}

			switch apiKind {
			case userconfig.RealtimeAPIKind.String():
				if err := realtimeapi.UpdateAutoscalerCron(&deployment, api); err != nil {
					operatorLogger.Fatal(errors.Wrap(err, "init"))
				}
			case userconfig.AsyncAPIKind.String():
				if err := asyncapi.UpdateMetricsCron(&deployment); err != nil {
					operatorLogger.Fatal(errors.Wrap(err, "init"))
				}

				if err := asyncapi.UpdateAutoscalerCron(&deployment, *api); err != nil {
					operatorLogger.Fatal(errors.Wrap(err, "init"))
				}
			}
		}
	}

	router := mux.NewRouter()

	routerWithoutAuth := router.NewRoute().Subrouter()
	routerWithoutAuth.Use(endpoints.PanicMiddleware)
	routerWithoutAuth.HandleFunc("/verifycortex", endpoints.VerifyCortex).Methods("GET")

	if config.Provider == types.AWSProviderType {
		routerWithoutAuth.HandleFunc("/batch/{apiName}", endpoints.SubmitBatchJob).Methods("POST")
		routerWithoutAuth.HandleFunc("/batch/{apiName}", endpoints.GetBatchJob).Methods("GET")
		routerWithoutAuth.HandleFunc("/batch/{apiName}", endpoints.StopBatchJob).Methods("DELETE")
	}
	routerWithoutAuth.HandleFunc("/tasks/{apiName}", endpoints.SubmitTaskJob).Methods("POST")
	routerWithoutAuth.HandleFunc("/tasks/{apiName}", endpoints.GetTaskJob).Methods("GET")
	routerWithoutAuth.HandleFunc("/tasks/{apiName}", endpoints.StopTaskJob).Methods("DELETE")

	// prometheus metrics
	routerWithoutAuth.Handle("/metrics", promhttp.Handler()).Methods("GET")

	routerWithAuth := router.NewRoute().Subrouter()

	routerWithAuth.Use(endpoints.PanicMiddleware)
	if config.Provider == types.AWSProviderType {
		routerWithAuth.Use(endpoints.AWSAuthMiddleware)
	}
	routerWithAuth.Use(endpoints.ClientIDMiddleware)
	routerWithAuth.Use(endpoints.APIVersionCheckMiddleware)

	routerWithAuth.HandleFunc("/info", endpoints.Info).Methods("GET")
	routerWithAuth.HandleFunc("/deploy", endpoints.Deploy).Methods("POST")
	routerWithAuth.HandleFunc("/patch", endpoints.Patch).Methods("POST")
	routerWithAuth.HandleFunc("/refresh/{apiName}", endpoints.Refresh).Methods("POST")
	routerWithAuth.HandleFunc("/delete/{apiName}", endpoints.Delete).Methods("DELETE")
	routerWithAuth.HandleFunc("/get", endpoints.GetAPIs).Methods("GET")
	routerWithAuth.HandleFunc("/get/{apiName}", endpoints.GetAPI).Methods("GET")
	routerWithAuth.HandleFunc("/get/{apiName}/{apiID}", endpoints.GetAPIByID).Methods("GET")
	routerWithAuth.HandleFunc("/logs/{apiName}", endpoints.ReadLogs)

	operatorLogger.Info("Running on port " + _operatorPortStr)
	operatorLogger.Fatal(http.ListenAndServe(":"+_operatorPortStr, router))
}
