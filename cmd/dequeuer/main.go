/*
Copyright 2022 Cortex Labs, Inc.

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
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/dequeuer"
	awslib "github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/logging"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/probe"
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"go.uber.org/zap"
)

func main() {
	var (
		clusterConfigPath string
		clusterUID        string
		probesPath        string
		queueURL          string
		userContainerPort int
		apiName           string
		jobID             string
		statsdAddress     string
		apiKind           string
		adminPort         int
		workers           int
	)
	flag.StringVar(&clusterConfigPath, "cluster-config", "", "cluster config path")
	flag.StringVar(&clusterUID, "cluster-uid", "", "cluster unique identifier")
	flag.StringVar(&probesPath, "probes-path", "", "path to the probes spec")
	flag.StringVar(&queueURL, "queue", "", "target queue URL from which the api messages will be dequeued")
	flag.StringVar(&apiKind, "api-kind", "", fmt.Sprintf("api kind (%s|%s)", userconfig.BatchAPIKind.String(), userconfig.AsyncAPIKind.String()))
	flag.StringVar(&apiName, "api-name", "", "api name")
	flag.StringVar(&jobID, "job-id", "", "job ID")
	flag.StringVar(&statsdAddress, "statsd-address", "", "address to push statsd metrics")
	flag.IntVar(&userContainerPort, "user-port", 8080, "target port to which the dequeued messages will be sent to")
	flag.IntVar(&adminPort, "admin-port", 0, "port where the admin server (for the probes) will be exposed")
	flag.IntVar(&workers, "workers", 1, "number of workers pulling from the queue")

	flag.Parse()

	version := os.Getenv("CORTEX_VERSION")
	if version == "" {
		version = consts.CortexVersion
	}

	log := logging.GetLogger()
	defer func() {
		_ = log.Sync()
	}()

	switch {
	case clusterConfigPath == "":
		log.Fatal("--cluster-config is a required option")
	case probesPath == "":
		log.Fatal("--probes-path is a required option")
	case queueURL == "":
		log.Fatal("--queue is a required option")
	case apiName == "":
		log.Fatal("--api-name is a required option")
	case apiKind == "":
		log.Fatal("--api-kind is a required option")
	case adminPort == 0:
		log.Fatal("--admin-port is a required option")
	}

	targetURL := "http://127.0.0.1:" + strconv.Itoa(userContainerPort)

	clusterConfig, err := clusterconfig.NewForFile(clusterConfigPath)
	if err != nil {
		exit(log, err)
	}

	awsClient, err := awslib.NewForRegion(clusterConfig.Region)
	if err != nil {
		exit(log, err, "failed to create aws client")
	}

	_, userID, err := awsClient.CheckCredentials()
	if err != nil {
		exit(log, err)
	}

	err = telemetry.Init(telemetry.Config{
		Enabled: clusterConfig.Telemetry,
		UserID:  userID,
		Properties: map[string]string{
			"kind":       apiKind,
			"image_type": "dequeuer",
		},
		Environment: "api",
		LogErrors:   true,
		BackoffMode: telemetry.BackoffDuplicateMessages,
	})
	if err != nil {
		log.Fatalw("failed to initialize telemetry", "error", err)
	}
	defer telemetry.Close()

	var probes []*probe.Probe
	if files.IsFile(probesPath) {
		probes, err = dequeuer.ProbesFromFile(probesPath, log)
		if err != nil {
			exit(log, err, fmt.Sprintf("unable to read probes from %s", probesPath))
		}
	}

	if !dequeuer.HasTCPProbeTargetingUserPod(probes, userContainerPort) {
		probes = append(probes, probe.NewDefaultProbe(fmt.Sprintf("http://localhost:%d", userContainerPort), log))
	}

	adminHandler := http.NewServeMux()
	adminHandler.Handle("/healthz", dequeuer.HealthcheckHandler(func() bool {
		return probe.AreProbesHealthy(probes)
	}))

	var dequeuerConfig dequeuer.SQSDequeuerConfig
	var messageHandler dequeuer.MessageHandler

	switch apiKind {
	case userconfig.BatchAPIKind.String():
		if jobID == "" {
			log.Fatal("--job-id is a required option")
		}

		config := dequeuer.BatchMessageHandlerConfig{
			Region:    clusterConfig.Region,
			APIName:   apiName,
			JobID:     jobID,
			QueueURL:  queueURL,
			TargetURL: targetURL,
		}

		metricsClient, err := statsd.New(statsdAddress)
		if err != nil {
			exit(log, err, "unable to initialize metrics client")
		}

		messageHandler = dequeuer.NewBatchMessageHandler(config, awsClient, metricsClient, log)
		dequeuerConfig = dequeuer.SQSDequeuerConfig{
			Region:           clusterConfig.Region,
			QueueURL:         queueURL,
			StopIfNoMessages: true,
			Workers:          workers,
		}

	case userconfig.AsyncAPIKind.String():
		if clusterUID == "" {
			log.Fatal("--cluster-uid is a required option")
		}

		config := dequeuer.AsyncMessageHandlerConfig{
			ClusterUID: clusterUID,
			Bucket:     clusterConfig.Bucket,
			APIName:    apiName,
			TargetURL:  targetURL,
		}

		asyncStatsReporter := dequeuer.NewAsyncPrometheusStatsReporter()
		messageHandler = dequeuer.NewAsyncMessageHandler(config, awsClient, asyncStatsReporter, log)
		dequeuerConfig = dequeuer.SQSDequeuerConfig{
			Region:           clusterConfig.Region,
			QueueURL:         queueURL,
			StopIfNoMessages: false,
			Workers:          workers,
		}

		// report prometheus metrics for async api kinds
		adminHandler.Handle("/metrics", asyncStatsReporter)
	default:
		exit(log, err, fmt.Sprintf("kind %s is not supported", apiKind))
	}

	errCh := make(chan error)

	go func() {
		server := &http.Server{
			Addr:    ":" + strconv.Itoa(adminPort),
			Handler: adminHandler,
		}
		log.Infof("Starting %s server on %s", consts.AdminPortName, server.Addr)
		errCh <- server.ListenAndServe()
	}()

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)

	sqsDequeuer, err := dequeuer.NewSQSDequeuer(dequeuerConfig, awsClient, log)
	if err != nil {
		exit(log, err, "failed to create sqs dequeuer")
	}

	go func() {
		log.Info("Starting dequeuer...")
		errCh <- sqsDequeuer.Start(messageHandler, func() bool {
			return probe.AreProbesHealthy(probes)
		})
	}()

	var stopChs []chan struct{}
	for _, p := range probes {
		stopChs = append(stopChs, p.StartProbing())
	}

	defer func() {
		for _, stopCh := range stopChs {
			stopCh <- struct{}{}
		}
	}()

	select {
	case err = <-errCh:
		exit(log, err, "error during message dequeueing or error from admin server")
	case <-sigint:
		log.Info("Received INT or TERM signal, handling a graceful shutdown...")
		sqsDequeuer.Shutdown()
		log.Info("Shutdown complete, exiting...")
	}
}

func exit(log *zap.SugaredLogger, err error, wrapStrs ...string) {
	if err == nil {
		os.Exit(0)
	}

	for _, str := range wrapStrs {
		err = errors.Wrap(err, str)
	}

	telemetry.Error(err)
	if !errors.IsNoPrint(err) {
		log.Fatal(err)
	}

	os.Exit(1)
}
