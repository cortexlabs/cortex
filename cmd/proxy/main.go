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
	"time"

	"github.com/cortexlabs/cortex/pkg/lib/logging"
	"github.com/cortexlabs/cortex/pkg/proxy"
)

const (
	_reportInterval        = 10 * time.Second
	_requestSampleInterval = 1 * time.Second
)

func main() {
	var (
		port              int
		userPort          int
		targetConcurrency int
		maxConcurrency    int
	)

	flag.IntVar(&port, "port", 8000, "port where the proxy will be served")
	flag.IntVar(&userPort, "user-port", 8080, "port where the proxy will redirect to the traffic to")
	flag.IntVar(&targetConcurrency, "target-concurrency", 0, "target concurrency for user container")
	flag.IntVar(&maxConcurrency, "max-concurrency", 0, "max concurrency for user container")
	flag.Parse()

	log := logging.GetLogger()
	defer func() {
		_ = log.Sync()
	}()

	switch {
	case targetConcurrency == 0:
		log.Fatal("--target-concurrency is required")
	case maxConcurrency == 0:
		maxConcurrency = targetConcurrency * 10
	}

	target := "127.0.0.1:" + strconv.Itoa(port)
	httpProxy := proxy.NewReverseProxy(target, maxConcurrency, maxConcurrency)

	stats := &proxy.RequestStats{}
	breaker := proxy.NewBreaker(
		proxy.BreakerParams{
			QueueDepth:      maxConcurrency,
			MaxConcurrency:  targetConcurrency,
			InitialCapacity: targetConcurrency,
		},
	)
	handler := proxy.ProxyHandler(breaker, httpProxy)

	go func() {
		reportTicker := time.NewTicker(_reportInterval)
		defer reportTicker.Stop()

		requestSamplingTicker := time.NewTicker(_requestSampleInterval)
		defer requestSamplingTicker.Stop()

		for {
			select {
			case <-reportTicker.C:
				go func() {
					stats.Report() // TODO: report on prometheus
				}()
			case <-requestSamplingTicker.C:
				go func() {
					stats.Append(breaker.InFlight())
				}()
			}
		}
	}()

	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(userPort), handler))
}
