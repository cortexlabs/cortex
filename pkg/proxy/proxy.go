/*
Copyright 2018 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Modifications Copyright 2022 Cortex Labs, Inc.
*/

package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

// NewReverseProxy creates a new cortex base reverse proxy
func NewReverseProxy(target string, maxIdle, maxIdlePerHost int) *httputil.ReverseProxy {
	targetURL, err := url.Parse(target)
	if err != nil {
		panic(err)
	}

	httpProxy := httputil.NewSingleHostReverseProxy(targetURL)
	httpProxy.Transport = buildHTTPTransport(maxIdle, maxIdlePerHost)

	return httpProxy
}

func buildHTTPTransport(maxIdle, maxIdlePerHost int) http.RoundTripper {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DisableKeepAlives = false
	transport.MaxIdleConns = maxIdle
	transport.MaxIdleConnsPerHost = maxIdlePerHost
	transport.ForceAttemptHTTP2 = false
	transport.DisableCompression = true
	return transport
}
