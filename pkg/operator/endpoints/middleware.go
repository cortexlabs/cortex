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

package endpoints

import (
	"context"
	"net/http"
	"strings"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/types"
)

var _cachedClientIDs = strset.New()

type ctxKey int

const (
	ctxKeyUnknown ctxKey = iota
	ctxKeyClient
)

func PanicMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer recoverAndRespond(w, r)
		next.ServeHTTP(w, r)
	})
}

func ClientIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if clientID := r.URL.Query().Get("clientID"); clientID != "" {
			// Add clientID to context
			ctx := context.WithValue(r.Context(), ctxKeyClient, clientID)
			r = r.WithContext(ctx)

			if !_cachedClientIDs.Has(clientID) {
				telemetry.RecordOperatorID(clientID, config.OperatorID())
				_cachedClientIDs.Add(clientID)
			}
		}
		next.ServeHTTP(w, r)
	})
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")

		if authHeader == "" {
			respondError(w, r, ErrorHeaderMissing("Authorization"))
			return
		}

		if len(authHeader) < 10 || !strings.HasPrefix(authHeader, "CortexAWS") {
			respondError(w, r, ErrorHeaderMalformed("Authorization"))
			return
		}

		parts := strings.Split(authHeader[10:], "|")
		if len(parts) != 2 {
			respondError(w, r, ErrorHeaderMalformed("Authorization"))
			return
		}

		if config.Provider == types.AWSProviderType {
			accessKeyID, secretAccessKey := parts[0], parts[1]
			awsClient, err := aws.NewFromCreds(*config.Cluster.Region, accessKeyID, secretAccessKey)
			if err != nil {
				respondError(w, r, ErrorAuthAPIError())
				return
			}

			accountID, _, err := awsClient.CheckCredentials()
			if err != nil {
				respondErrorCode(w, r, http.StatusForbidden, ErrorAuthInvalid())
				return
			}

			operatorAccountID, _, err := config.AWS.GetCachedAccountID()
			if err != nil {
				respondError(w, r, ErrorAuthAPIError())
				return
			}

			if accountID != operatorAccountID {
				respondErrorCode(w, r, http.StatusForbidden, ErrorAuthOtherAccount())
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func APIVersionCheckMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/info" {
			next.ServeHTTP(w, r)
			return
		}

		clientVersion := r.Header.Get("CortexAPIVersion")
		if clientVersion == "" {
			respondError(w, r, ErrorHeaderMissing("CortexAPIVersion"))
			return
		}

		if clientVersion != consts.CortexVersion {
			respondError(w, r, ErrorAPIVersionMismatch(consts.CortexVersion, clientVersion))
			return
		}
		next.ServeHTTP(w, r)
	})
}
