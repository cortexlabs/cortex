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
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
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

	transport.ExpectContinueTimeout = 60 * time.Second
	transport.IdleConnTimeout = 60 * time.Second
	transport.ResponseHeaderTimeout = 60 * time.Second
	transport.TLSHandshakeTimeout = 60 * time.Second

	transport.Dial = nil
	// transport.DialContext = nil
	transport.DialContext = (&net.Dialer{
		Timeout:   60 * time.Second,
		KeepAlive: 15 * time.Second,
	}).DialContext

	return transport
}
