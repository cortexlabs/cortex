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
	"net/http"
	"strings"
)

func ProxyHandler(stats *RequestStats, next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if isKubeletProbe(r) {
			next.ServeHTTP(w, r)
			return
		}

		stats.IncomingRequestEvent()
		defer stats.OutgoingRequestEvent()

		next.ServeHTTP(w, r)
	}
}

func isKubeletProbe(r *http.Request) bool {
	return strings.HasPrefix(r.Header.Get(_userAgentKey), _kubeProbeUserAgentPrefix) ||
		r.Header.Get(_kubeletProbeHeaderName) != ""
}
