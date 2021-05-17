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

package proxy

import (
	"context"
	"errors"
	"net/http"
	"strings"
)

func Handler(breaker *Breaker, next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if isKubeletProbe(r) || breaker == nil {
			next.ServeHTTP(w, r)
			return
		}

		if err := breaker.Maybe(r.Context(), func() {
			next.ServeHTTP(w, r)
		}); err != nil {
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, ErrRequestQueueFull) {
				http.Error(w, err.Error(), http.StatusServiceUnavailable)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
		}
	}
}

func isKubeletProbe(r *http.Request) bool {
	return strings.HasPrefix(r.Header.Get(_userAgentKey), _kubeProbeUserAgentPrefix) ||
		r.Header.Get(_kubeletProbeHeaderName) != ""
}
