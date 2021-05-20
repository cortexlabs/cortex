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
	"strings"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/dequeuer"
	awslib "github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/errors"

	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func createLogger() (*zap.Logger, error) {
	logLevelEnv := strings.ToUpper(os.Getenv("CORTEX_LOG_LEVEL"))
	disableJSONLogging := os.Getenv("CORTEX_DISABLE_JSON_LOGGING")

	var logLevelZap zapcore.Level
	switch logLevelEnv {
	case "DEBUG":
		logLevelZap = zapcore.DebugLevel
	case "WARNING":
		logLevelZap = zapcore.WarnLevel
	case "ERROR":
		logLevelZap = zapcore.ErrorLevel
	default:
		logLevelZap = zapcore.InfoLevel
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.MessageKey = "message"

	encoding := "json"
	if strings.ToLower(disableJSONLogging) == "true" {
		encoding = "console"
	}

	return zap.Config{
		Level:            zap.NewAtomicLevelAt(logLevelZap),
		Encoding:         encoding,
		EncoderConfig:    encoderConfig,
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}.Build()
}

func main() {
	var (
		region     string
		queueURL   string
		apiName    string
		jobID      string
		statsdPort int
		apiKind    string
	)
	flag.StringVar(&region, "region", os.Getenv("CORTEX_REGION"), "cluster region (can be set throught the CORTEX_REGION env variable)")
	flag.StringVar(&queueURL, "queue", "", "target queue URL from which the api messages will be dequeued")
	flag.StringVar(&apiName, "apiName", "", "api name")
	flag.StringVar(&jobID, "jobID", "", "job ID")
	flag.StringVar(&apiKind, "apiKind", "", fmt.Sprintf("api kind (%s|%s)", userconfig.BatchAPIKind.String(), userconfig.AsyncAPIKind.String()))
	flag.IntVar(&statsdPort, "statsdPort", 9125, "port for to send udp statsd metrics")

	flag.Parse()

	version := os.Getenv("CORTEX_VERSION")
	if version == "" {
		version = consts.CortexVersion
	}

	hostIP := os.Getenv("HOST_IP")

	log, err := createLogger()
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = log.Sync()
	}()

	awsClient, err := awslib.NewForRegion(region)
	if err != nil {
		panic(err)
	}

	switch {
	case region == "":
		log.Fatal("-region is a required option")
	case queueURL == "":
		log.Fatal("-queue is a required option")
	case apiName == "":
		log.Fatal("-apiName is a required option")
	case jobID == "":
		log.Fatal("-jobID is a required option")
	case apiKind == "":
		log.Fatal("-apiKind is a required option")
	}

	switch apiKind {
	case userconfig.BatchAPIKind.String():
		config := dequeuer.BatchMessageHandlerConfig{
			Region:   region,
			APIName:  apiName,
			JobID:    jobID,
			QueueURL: queueURL,
		}

		metricsClient, err := statsd.New(fmt.Sprintf("%s:%d", hostIP, statsdPort))
		if err != nil {
			panic(errors.Wrap(err, "unable to initialize metrics client"))
		}
		messageHandler := dequeuer.NewBatchMessageHandler(config, awsClient, metricsClient, log.Sugar())

		handlerConfig := dequeuer.SQSDequeuerConfig{
			Region:           region,
			QueueURL:         queueURL,
			StopIfNoMessages: true,
		}

		sqsDequeuer, err := dequeuer.NewSQSDequeuer(handlerConfig, awsClient, log.Sugar())
		if err != nil {
			log.Fatal("failed to create sqs handler", zap.Error(err))
		}

		if err = sqsDequeuer.Start(messageHandler); err != nil {
			log.Fatal("error durring message dequeueing", zap.Error(err))
		}
	default:
		log.Sugar().Fatalf("kind %s is not supported", apiKind)
	}
}
