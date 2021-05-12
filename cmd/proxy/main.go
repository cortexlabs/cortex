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

package main

import (
	"flag"
	"net/http"
	"strconv"

	"github.com/cortexlabs/cortex/pkg/lib/logging"
	"github.com/cortexlabs/cortex/pkg/proxy"
)

func main() {
	var (
		port     int
		userPort int
	)

	flag.IntVar(&port, "port", 8000, "port where the proxy will be served")
	flag.IntVar(&userPort, "user-port", 8080, "port where the proxy will redirect to the traffic to")
	flag.Parse()

	log := logging.GetLogger()
	defer func() {
		_ = log.Sync()
	}()

	target := "127.0.0.1:" + strconv.Itoa(port)
	httpProxy := proxy.NewReverseProxy(target)

	stats := &proxy.RequestStats{}
	handler := proxy.ProxyHandler(stats, &httpProxy)

	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(userPort), handler))
}
