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
	"net/http/httputil"
)

// NewReverseProxy creates a new cortex base reverse proxy
func NewReverseProxy(target string) httputil.ReverseProxy {
	return httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = _proxyScheme
			req.URL.Host = target

			if _, ok := req.Header[_userAgentKey]; !ok {
				// explicitly disable User-Agent so it's not set to default value
				req.Header.Set(_userAgentKey, "")
			}
		},
	}
}
