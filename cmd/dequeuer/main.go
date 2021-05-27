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
	"fmt"
	"os"
	"os/signal"
	"strconv"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/dequeuer"
	awslib "github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/logging"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"go.uber.org/zap"
)

func main() {
	var (
		clusterConfigPath string
		clusterUID        string
		queueURL          string
		userContainerPort int
		apiName           string
		jobID             string
		statsdPort        int
		apiKind           string
	)
	flag.StringVar(&clusterConfigPath, "cluster-config", "", "cluster config path")
	flag.StringVar(&clusterUID, "cluster-uid", "", "cluster unique identifier")
	flag.StringVar(&queueURL, "queue", "", "target queue URL from which the api messages will be dequeued")
	flag.StringVar(&apiKind, "api-kind", "", fmt.Sprintf("api kind (%s|%s)", userconfig.BatchAPIKind.String(), userconfig.AsyncAPIKind.String()))
	flag.StringVar(&apiName, "api-name", "", "api name")
	flag.StringVar(&jobID, "job-id", "", "job ID")
	flag.IntVar(&userContainerPort, "user-port", 8080, "target port from which the dequeued messages will be sent to")
	flag.IntVar(&statsdPort, "statsd-port", 9125, "port for to send udp statsd metrics")

	flag.Parse()

	version := os.Getenv("CORTEX_VERSION")
	if version == "" {
		version = consts.CortexVersion
	}

	hostIP := os.Getenv("HOST_IP")

	log := logging.GetLogger()
	defer func() {
		_ = log.Sync()
	}()

	switch {
	case clusterConfigPath == "":
		log.Fatal("--cluster-config is a required option")
	case queueURL == "":
		log.Fatal("--queue is a required option")
	case apiName == "":
		log.Fatal("--api-name is a required option")
	case apiKind == "":
		log.Fatal("--api-kind is a required option")
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

	metricsClient, err := statsd.New(fmt.Sprintf("%s:%d", hostIP, statsdPort))
	if err != nil {
		exit(log, err, "unable to initialize metrics client")
	}

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

		messageHandler = dequeuer.NewBatchMessageHandler(config, awsClient, metricsClient, log)
		dequeuerConfig = dequeuer.SQSDequeuerConfig{
			Region:           clusterConfig.Region,
			QueueURL:         queueURL,
			StopIfNoMessages: true,
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

		messageHandler = dequeuer.NewAsyncMessageHandler(config, awsClient, log)
		dequeuerConfig = dequeuer.SQSDequeuerConfig{
			Region:           clusterConfig.Region,
			QueueURL:         queueURL,
			StopIfNoMessages: false,
		}
	default:
		exit(log, err, fmt.Sprintf("kind %s is not supported", apiKind))
	}

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt)

	sqsDequeuer, err := dequeuer.NewSQSDequeuer(dequeuerConfig, awsClient, log)
	if err != nil {
		exit(log, err, "failed to create sqs dequeuer")
	}

	errCh := make(chan error)
	go func() {
		log.Info("Starting dequeuer...")
		errCh <- sqsDequeuer.Start(messageHandler)
	}()

	select {
	case err = <-errCh:
		exit(log, err, "error during message dequeueing")
	case <-sigint:
		log.Info("Received TERM signal, handling a graceful shutdown...")
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

	if !errors.IsNoTelemetry(err) {
		telemetry.Error(err)
	}

	if !errors.IsNoPrint(err) {
		log.Error(err)
	}

	os.Exit(1)
}
