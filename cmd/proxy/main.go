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
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/cortexlabs/cortex/pkg/lib/logging"
	"github.com/cortexlabs/cortex/pkg/proxy"
	"github.com/cortexlabs/cortex/pkg/proxy/probe"
	"go.uber.org/zap"
)

const (
	_reportInterval        = 10 * time.Second
	_requestSampleInterval = 1 * time.Second
)

func main() {
	var (
		port              int
		metricsPort       int
		probePort         int
		userContainerPort int
		maxConcurrency    int
		maxQueueLength    int
	)

	flag.IntVar(&port, "port", 8000, "port where the proxy server will be exposed")
	flag.IntVar(&metricsPort, "metrics-port", 8001, "port where the metrics server will be exposed")
	flag.IntVar(&probePort, "probe-port", 8002, "port where the probe server will be exposed")
	flag.IntVar(&userContainerPort, "user-port", 8080, "port where the proxy will redirect to the traffic to")
	flag.IntVar(&maxConcurrency, "max-concurrency", 0, "max concurrency allowed for user container")
	flag.IntVar(&maxQueueLength, "max-queue-length", 0, "max request queue length for user container")
	flag.Parse()

	log := logging.GetLogger()
	defer func() {
		_ = log.Sync()
	}()

	switch {
	case maxConcurrency == 0:
		log.Fatal("--max-concurrency flag is required")
	case maxQueueLength == 0:
		log.Fatal("--max-queue-length flag is required")
	}

	target := "http://127.0.0.1:" + strconv.Itoa(userContainerPort)
	httpProxy := proxy.NewReverseProxy(target, maxQueueLength, maxQueueLength)

	requestCounterStats := &proxy.RequestStats{}
	breaker := proxy.NewBreaker(
		proxy.BreakerParams{
			QueueDepth:      maxQueueLength,
			MaxConcurrency:  maxConcurrency,
			InitialCapacity: maxConcurrency,
		},
	)

	promStats := proxy.NewPrometheusStatsReporter()
	readinessProbe := probe.NewDefaultProbe(log, target) // TODO: initialize custom probe from flags

	go func() {
		reportTicker := time.NewTicker(_reportInterval)
		defer reportTicker.Stop()

		requestSamplingTicker := time.NewTicker(_requestSampleInterval)
		defer requestSamplingTicker.Stop()

		for {
			select {
			case <-reportTicker.C:
				go func() {
					report := requestCounterStats.Report()
					promStats.Report(report)
				}()
			case <-requestSamplingTicker.C:
				go func() {
					requestCounterStats.Append(breaker.InFlight())
				}()
			}
		}
	}()

	servers := map[string]*http.Server{
		"proxy": {
			Addr:    ":" + strconv.Itoa(port),
			Handler: proxy.Handler(breaker, httpProxy),
		},
		"metrics": {
			Addr:    ":" + strconv.Itoa(metricsPort),
			Handler: promStats,
		},
		"probe": {
			Addr:    ":" + strconv.Itoa(probePort),
			Handler: probe.Handler(readinessProbe),
		},
	}

	errCh := make(chan error)
	for name, server := range servers {
		go func(name string, server *http.Server) {
			log.Infof("Starting %s server on %s", name, server.Addr)
			errCh <- server.ListenAndServe()
		}(name, server)
	}

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt)

	select {
	case err := <-errCh:
		log.Fatal("failed to start proxy server", zap.Error(err))
	case <-sigint:
		// We received an interrupt signal, shut down.
		log.Info("Received TERM signal, handling a graceful shutdown...")

		for name, server := range servers {
			log.Infof("Shutting down %s server", name)
			if err := server.Shutdown(context.Background()); err != nil {
				// Error from closing listeners, or context timeout:
				log.Warn("HTTP server Shutdown Error", zap.Error(err))
			}
		}
		log.Info("Shutdown complete, exiting...")
	}
}
