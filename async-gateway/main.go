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
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	_defaultPort = "8080"
)

func createLogger() (*zap.Logger, error) {
	logLevelEnv := os.Getenv("CORTEX_LOG_LEVEL")
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

// usage: ./gateway -bucket <bucket> -region <region> -port <port> -queue queue <apiName>
func main() {
	log, err := createLogger()
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = log.Sync()
	}()

	var (
		port        = flag.String("port", _defaultPort, "port on which the gateway server runs on")
		queueURL    = flag.String("queue", "", "SQS queue URL")
		region      = flag.String("region", "", "AWS region")
		bucket      = flag.String("bucket", "", "AWS bucket")
		clusterName = flag.String("cluster", "", "cluster name")
	)
	flag.Parse()

	switch {
	case *queueURL == "":
		log.Fatal("missing required option: -queue")
	case *region == "":
		log.Fatal("missing required option: -region")
	case *bucket == "":
		log.Fatal("missing required option: -bucket")
	case *clusterName == "":
		log.Fatal("missing required option: -cluster")
	}

	apiName := flag.Arg(0)
	if apiName == "" {
		log.Fatal("apiName argument was not provided")
	}

	sess, err := session.NewSessionWithOptions(session.Options{
		Config: aws.Config{
			Region: region,
		},
		SharedConfigState: session.SharedConfigEnable,
	})
	if err != nil {
		log.Fatal("failed to create AWS session: %s", zap.Error(err))
	}

	s3Storage := NewS3(sess, *bucket)

	sqsQueue := NewSQS(*queueURL, sess)

	svc := NewService(*clusterName, apiName, sqsQueue, s3Storage, log)
	ep := NewEndpoint(svc, log)

	router := mux.NewRouter()
	router.HandleFunc("/", ep.CreateWorkload).Methods("POST")
	router.HandleFunc(
		"/healthz",
		func(w http.ResponseWriter, r *http.Request) {
			respondPlainText(w, http.StatusOK, "ok")
		},
	)
	router.HandleFunc("/{id}", ep.GetWorkload).Methods("GET")

	log.Info("Running on port " + *port)
	if err = http.ListenAndServe(":"+*port, router); err != nil {
		log.Fatal("failed to start server", zap.Error(err))
	}
}
