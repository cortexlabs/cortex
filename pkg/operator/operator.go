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

package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	cron "gopkg.in/robfig/cron.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/operator/argo"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/operator/context"
	"github.com/cortexlabs/cortex/pkg/operator/endpoints"
	"github.com/cortexlabs/cortex/pkg/operator/spark"
	"github.com/cortexlabs/cortex/pkg/operator/workloads"
)

const (
	operatorPortStr       = "8888"
	workflowDeletionDelay = 60 // seconds
	cronInterval          = 5  // seconds
)

var (
	awsClient       *aws.Client
	telemtryClient  *telemetry.Client
	markedWorkflows = strset.New()
)

func main() {
	conf := config.NewFromEnv()
	config.AWS = aws.New(conf.Region, conf.Bucket)
	config.Telemetry = telemetry.New(conf.TelemetryURL, config.AWS.HashedAccountID)

	var err error
	if config.Kubernetes, err = k8s.NewClient(conf.Namespace, conf.OperatorInCluster); err != nil {
		config.Telemetry.ReportErrorBlocking(err)
		errors.Exit(err)
	}

	argo.Init()

	if err := context.Init(); err != nil {
		config.Telemetry.ReportErrorBlocking(err)
		errors.Exit(err)
	}

	if err := spark.Init(); err != nil {
		config.Telemetry.ReportErrorBlocking(err)
		errors.Exit(err)
	}

	if err := workloads.Init(); err != nil {
		config.Telemetry.ReportErrorBlocking(err)
		errors.Exit(err)
	}

	config.Telemetry.ReportEvent("operator.init")
	startCron()

	router := mux.NewRouter()
	router.Use(panicMiddleware)
	router.Use(apiVersionCheckMiddleware)
	router.Use(authMiddleware)

	router.HandleFunc("/deploy", endpoints.Deploy).Methods("POST")
	router.HandleFunc("/delete", endpoints.Delete).Methods("POST")
	router.HandleFunc("/resources", endpoints.GetResources).Methods("GET")
	router.HandleFunc("/aggregate/{id}", endpoints.GetAggregate).Methods("GET")
	router.HandleFunc("/logs/read", endpoints.ReadLogs)

	log.Print("Running on port " + operatorPortStr)
	log.Fatal(http.ListenAndServe(":"+operatorPortStr, router))
}

func panicMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer endpoints.RecoverAndRespond(w)
		next.ServeHTTP(w, r)
	})
}

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")

		if !strings.HasPrefix(authHeader, "CortexAWS") {
			endpoints.RespondError(w, endpoints.ErrorAuthHeaderMissing())
			return
		}

		parts := strings.Split(authHeader[10:], "|")
		if len(parts) != 2 {
			endpoints.RespondError(w, endpoints.ErrorAuthHeaderMalformed())
			return
		}

		accessKeyID, secretAccessKey := parts[0], parts[1]
		authed, err := config.AWS.AuthUser(accessKeyID, secretAccessKey)
		if err != nil {
			endpoints.RespondError(w, endpoints.ErrorAuthAPIError())
			return
		}

		if !authed {
			endpoints.RespondErrorCode(w, http.StatusForbidden, endpoints.ErrorAuthForbidden())
			return
		}

		next.ServeHTTP(w, r)
	})
}

func apiVersionCheckMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientVersion := r.Header.Get("CortexAPIVersion")
		if clientVersion != consts.CortexVersion {
			endpoints.RespondError(w, ErrorAPIVersionMismatch(consts.CortexVersion, clientVersion))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func startCron() {
	cronRunner := cron.New()
	cronInterval := fmt.Sprintf("@every %ds", cronInterval)
	cronRunner.AddFunc(cronInterval, runCron)
	cronRunner.Start()
}

func runCron() {
	defer reportAndRecover("cron failed")
	apiPods, err := k8s.ListPodsByLabels(map[string]string{
		"workloadType": workloads.WorkloadTypeAPI,
		"userFacing":   "true",
	})
	if err != nil {
		config.Telemetry.ReportError(err)
		errors.PrintError(err)
	}

	if err := workloads.UpdateAPISavedStatuses(apiPods); err != nil {
		config.Telemetry.ReportError(err)
		errors.PrintError(err)
	}

	if err := workloads.UploadLogPrefixesFromAPIPods(apiPods); err != nil {
		config.Telemetry.ReportError(err)
		errors.PrintError(err)
	}

	failedPods, err := k8s.ListPods(&metav1.ListOptions{
		FieldSelector: "status.phase=Failed",
	})
	if err != nil {
		config.Telemetry.ReportError(err)
		errors.PrintError(err)
	}

	if err := workloads.UpdateDataWorkflowErrors(failedPods); err != nil {
		config.Telemetry.ReportError(err)
		errors.PrintError(err)
	}
}

func deleteWorkflowDelayed(wfName string) {
	deletionDelay := time.Duration(workflowDeletionDelay) * time.Second
	if !markedWorkflows.Has(wfName) {
		markedWorkflows.Add(wfName)
		time.Sleep(deletionDelay)
		argo.Delete(wfName)
		go deleteMarkerDelayed(markedWorkflows, wfName)
	}
}

// Wait some time before trying to delete again
func deleteMarkerDelayed(markerMap strset.Set, key string) {
	time.Sleep(20 * time.Second)
	markerMap.Remove(key)
}

func reportAndRecover(strs ...string) error {
	if errInterface := recover(); errInterface != nil {
		err := errors.CastRecoverError(errInterface, strs...)
		config.Telemetry.ReportError(err)
		errors.PrintError(err)
		return err
	}
	return nil
}
