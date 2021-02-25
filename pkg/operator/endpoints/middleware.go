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

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/operator/config"
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
				telemetry.RecordOperatorID(clientID, config.OperatorMetadata.OperatorID)
				_cachedClientIDs.Add(clientID)
			}
		}
		next.ServeHTTP(w, r)
	})
}

func AWSAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get(consts.AuthHeader)

		if authHeader == "" {
			respondError(w, r, ErrorHeaderMissing(consts.AuthHeader))
			return
		}

		accountID, err := aws.ExecuteIdentityRequestFromHeader(authHeader)
		if err != nil {
			respondError(w, r, err)
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
