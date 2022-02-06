/*
Copyright 2022 Cortex Labs, Inc.

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

	"github.com/cortexlabs/cortex/pkg/config"
	"github.com/cortexlabs/cortex/pkg/lib/cron"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/logging"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/operator/endpoints"
	"github.com/cortexlabs/cortex/pkg/operator/lib/exit"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/operator/resources/asyncapi"
	"github.com/cortexlabs/cortex/pkg/operator/resources/job/taskapi"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var operatorLogger = logging.GetLogger()

const _operatorPortStr = "8888"

func main() {
	if err := config.Init(); err != nil {
		exit.ErrorNoTelemetry(errors.Wrap(err, "init"))
	}

	telemetry.Event("operator.init")

	cron.Run(operator.DeleteEvictedPods, operator.ErrorHandler("delete evicted pods"), time.Hour)
	cron.Run(operator.ClusterTelemetry, operator.ErrorHandler("instance telemetry"), 1*time.Hour)
	cron.Run(operator.CostBreakdown, operator.ErrorHandler("cost breakdown metrics"), 5*time.Minute)

	_, err := operator.UpdateMemoryCapacityConfigMap()
	if err != nil {
		exit.Error(errors.Wrap(err, "init"))
	}

	cron.Run(taskapi.ManageJobResources, operator.ErrorHandler("manage task jobs"), taskapi.ManageJobResourcesCronPeriod)

	deployments, err := config.K8s.ListDeploymentsWithLabelKeys("apiName")
	if err != nil {
		exit.Error(errors.Wrap(err, "init"))
	}

	for i := range deployments {
		deployment := deployments[i]
		apiKind := deployment.Labels["apiKind"]
		switch apiKind {
		case userconfig.AsyncAPIKind.String():
			if err := asyncapi.UpdateAPIMetricsCron(&deployment); err != nil {
				operatorLogger.Fatal(errors.Wrap(err, "init"))
			}
		}
	}

	router := mux.NewRouter()

	routerWithoutAuth := router.NewRoute().Subrouter()
	routerWithoutAuth.Use(endpoints.PanicMiddleware)
	routerWithoutAuth.HandleFunc("/verifycortex", endpoints.VerifyCortex).Methods("GET")

	routerWithoutAuth.HandleFunc("/batch/{apiName}", endpoints.SubmitBatchJob).Methods("POST")
	routerWithoutAuth.HandleFunc("/batch/{apiName}", endpoints.GetBatchJob).Methods("GET")
	routerWithoutAuth.HandleFunc("/batch/{apiName}", endpoints.StopBatchJob).Methods("DELETE")
	routerWithoutAuth.HandleFunc("/tasks/{apiName}", endpoints.SubmitTaskJob).Methods("POST")
	routerWithoutAuth.HandleFunc("/tasks/{apiName}", endpoints.GetTaskJob).Methods("GET")
	routerWithoutAuth.HandleFunc("/tasks/{apiName}", endpoints.StopTaskJob).Methods("DELETE")

	// prometheus metrics
	routerWithoutAuth.Handle("/metrics", promhttp.Handler()).Methods("GET")

	routerWithAuth := router.NewRoute().Subrouter()

	routerWithAuth.Use(endpoints.PanicMiddleware)
	routerWithAuth.Use(endpoints.APIVersionCheckMiddleware)
	routerWithAuth.Use(endpoints.AWSAuthMiddleware)
	routerWithAuth.Use(endpoints.ClientIDMiddleware)

	routerWithAuth.HandleFunc("/info", endpoints.Info).Methods("GET")
	routerWithAuth.HandleFunc("/deploy", endpoints.Deploy).Methods("POST")
	routerWithAuth.HandleFunc("/refresh/{apiName}", endpoints.Refresh).Methods("POST")
	routerWithAuth.HandleFunc("/delete/{apiName}", endpoints.Delete).Methods("DELETE")
	routerWithAuth.HandleFunc("/get", endpoints.GetAPIs).Methods("GET")
	routerWithAuth.HandleFunc("/get/{apiName}", endpoints.GetAPI).Methods("GET")
	routerWithAuth.HandleFunc("/get/{apiName}/{apiID}", endpoints.GetAPIByID).Methods("GET")
	routerWithAuth.HandleFunc("/describe/{apiName}", endpoints.DescribeAPI).Methods("GET")
	routerWithAuth.HandleFunc("/streamlogs/{apiName}", endpoints.ReadLogs)
	routerWithAuth.HandleFunc("/logs/{apiName}", endpoints.GetLogURL).Methods("GET")

	operatorLogger.Info("Running on port " + _operatorPortStr)

	// inspired by our nginx config
	corsOptions := []handlers.CORSOption{
		handlers.AllowedOrigins([]string{"*"}),
		// custom headers are not supported currently, since "*" is not supported in AllowedHeaders(); here are some common ones:
		handlers.AllowedHeaders([]string{"Content-Type", "X-Requested-With", "User-Agent", "Accept", "Accept-Language", "Content-Language", "Origin"}),
		handlers.AllowedMethods([]string{"GET", "HEAD", "POST", "PUT", "OPTIONS"}),
		handlers.ExposedHeaders([]string{"Content-Length", "Content-Range"}),
		handlers.AllowCredentials(),
	}

	operatorLogger.Fatal(http.ListenAndServe(":"+_operatorPortStr, handlers.CORS(corsOptions...)(router)))
}
