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
	"log"
	"net/http"
	"strings"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/operator/endpoints"
	"github.com/cortexlabs/cortex/pkg/operator/workloads"
	"github.com/gorilla/mux"
)

const operatorPortStr = "8888"

var _cachedClientIDs = strset.New()

func main() {
	if err := config.Init(); err != nil {
		exit.Error(err)
	}

	telemetry.InternalEvent("operator.init")

	if err := workloads.Init(); err != nil {
		exit.Error(err)
	}

	router := mux.NewRouter()
	router.Use(panicMiddleware)
	router.Use(clientIDMiddleware)
	router.Use(apiVersionCheckMiddleware)
	router.Use(authMiddleware)

	router.HandleFunc("/info", endpoints.Info).Methods("GET")
	router.HandleFunc("/deploy", endpoints.Deploy).Methods("POST")
	router.HandleFunc("/delete", endpoints.Delete).Methods("POST")
	router.HandleFunc("/deployments", endpoints.GetDeployments).Methods("GET")
	router.HandleFunc("/metrics", endpoints.GetMetrics).Methods("GET")
	router.HandleFunc("/resources", endpoints.GetResources).Methods("GET")
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

func clientIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if clientID := r.URL.Query().Get("clientID"); clientID != "" {
			if !_cachedClientIDs.Has(clientID) {
				telemetry.RecordOperatorID(clientID, config.AWS.HashedAccountID)
				_cachedClientIDs.Add(clientID)
			}
		}
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
		userAccountID, validCreds, err := aws.AccountID(accessKeyID, secretAccessKey, *config.Cluster.Region)
		if err != nil {
			endpoints.RespondError(w, endpoints.ErrorAuthAPIError())
			return
		}
		if !validCreds {
			endpoints.RespondErrorCode(w, http.StatusForbidden, endpoints.ErrorAuthInvalid())
			return
		}
		if userAccountID != config.AWS.AccountID {
			endpoints.RespondErrorCode(w, http.StatusForbidden, endpoints.ErrorAuthOtherAccount())
			return
		}

		next.ServeHTTP(w, r)
	})
}

func apiVersionCheckMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/info" {
			next.ServeHTTP(w, r)
			return
		}

		clientVersion := r.Header.Get("CortexAPIVersion")
		if clientVersion != consts.CortexVersion {
			endpoints.RespondError(w, ErrorAPIVersionMismatch(consts.CortexVersion, clientVersion))
			return
		}
		next.ServeHTTP(w, r)
	})
}
