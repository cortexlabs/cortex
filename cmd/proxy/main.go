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

	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/logging"
	"github.com/cortexlabs/cortex/pkg/lib/probe"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/proxy"
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"go.uber.org/zap"
)

const (
	_reportInterval        = 10 * time.Second
	_requestSampleInterval = 1 * time.Second
)

func main() {
	var (
		port              int
		adminPort         int
		userContainerPort int
		maxConcurrency    int
		maxQueueLength    int
		clusterConfigPath string
	)

	flag.IntVar(&port, "port", 8000, "port where the proxy server will be exposed")
	flag.IntVar(&adminPort, "admin-port", 15000, "port where the admin server (for metrics and probes) will be exposed")
	flag.IntVar(&userContainerPort, "user-port", 8080, "port where the proxy will redirect to the traffic to")
	flag.IntVar(&maxConcurrency, "max-concurrency", 0, "max concurrency allowed for user container")
	flag.IntVar(&maxQueueLength, "max-queue-length", 0, "max request queue length for user container")
	flag.StringVar(&clusterConfigPath, "cluster-config", "", "cluster config path")
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
	case clusterConfigPath == "":
		log.Fatal("--cluster-config flag is required")
	}

	clusterConfig, err := clusterconfig.NewForFile(clusterConfigPath)
	if err != nil {
		exit(log, err)
	}

	awsClient, err := aws.NewForRegion(clusterConfig.Region)
	if err != nil {
		exit(log, err)
	}

	_, userID, err := awsClient.CheckCredentials()
	if err != nil {
		exit(log, err)
	}

	err = telemetry.Init(telemetry.Config{
		Enabled: clusterConfig.Telemetry,
		UserID:  userID,
		Properties: map[string]string{
			"kind":       userconfig.RealtimeAPIKind.String(),
			"image_type": "proxy",
		},
		Environment: "api",
		LogErrors:   true,
		BackoffMode: telemetry.BackoffDuplicateMessages,
	})
	if err != nil {
		exit(log, err)
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
	readinessProbe := probe.NewDefaultProbe(target, log)
	probeStopper := readinessProbe.StartProbing()
	defer func() {
		probeStopper <- true
	}()

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

	adminHandler := http.NewServeMux()
	adminHandler.Handle("/metrics", promStats)
	adminHandler.Handle("/healthz", probe.Handler(readinessProbe))

	servers := map[string]*http.Server{
		"proxy": {
			Addr:    ":" + strconv.Itoa(port),
			Handler: proxy.Handler(breaker, httpProxy),
		},
		"admin": {
			Addr:    ":" + strconv.Itoa(adminPort),
			Handler: adminHandler,
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
	case err = <-errCh:
		exit(log, errors.Wrap(err, "failed to start proxy server"))
	case <-sigint:
		// We received an interrupt signal, shut down.
		log.Info("Received TERM signal, handling a graceful shutdown...")

		for name, server := range servers {
			log.Infof("Shutting down %s server", name)
			if err := server.Shutdown(context.Background()); err != nil {
				// Error from closing listeners, or context timeout:
				log.Warn("HTTP server Shutdown Error", zap.Error(err))
				telemetry.Error(errors.Wrap(err, "HTTP server Shutdown Error"))
			}
		}
		log.Info("Shutdown complete, exiting...")
		telemetry.Close()
	}
}

func exit(log *zap.SugaredLogger, err error, wrapStrs ...string) {
	for _, str := range wrapStrs {
		err = errors.Wrap(err, str)
	}

	if err != nil && !errors.IsNoTelemetry(err) {
		telemetry.Error(err)
	}

	if err != nil && !errors.IsNoPrint(err) {
		log.Error(err)
	}

	telemetry.Close()
	os.Exit(1)
}
