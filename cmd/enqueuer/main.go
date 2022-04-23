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
	"os"
	"strings"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/enqueuer"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func createLogger() (*zap.Logger, error) {
	logLevelEnv := strings.ToLower(os.Getenv("CORTEX_LOG_LEVEL"))
	disableJSONLogging := os.Getenv("CORTEX_DISABLE_JSON_LOGGING")

	var logLevelZap zapcore.Level
	switch logLevelEnv {
	case "debug":
		logLevelZap = zapcore.DebugLevel
	case "warning":
		logLevelZap = zapcore.WarnLevel
	case "error":
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
		clusterUID string
		region     string
		bucket     string
		queueURL   string
		apiName    string
		jobID      string
	)
	flag.StringVar(&clusterUID, "cluster-uid", os.Getenv("CORTEX_CLUSTER_UID"), "cluster UID (can be set throught the CORTEX_CLUSTER_UID env variable)")
	flag.StringVar(&region, "region", os.Getenv("CORTEX_REGION"), "cluster region (can be set throught the CORTEX_REGION env variable)")
	flag.StringVar(&bucket, "bucket", os.Getenv("CORTEX_BUCKET"), "cortex S3 bucket (can be set throught the CORTEX_BUCKET env variable)")
	flag.StringVar(&queueURL, "queue", "", "target queue URL to where the api messages will be enqueued")
	flag.StringVar(&apiName, "apiName", "", "api name")
	flag.StringVar(&jobID, "jobID", "", "job ID")

	flag.Parse()

	version := os.Getenv("CORTEX_VERSION")
	if version == "" {
		version = consts.CortexVersion
	}

	log, err := createLogger()
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = log.Sync()
	}()

	switch {
	case clusterUID == "":
		log.Fatal("-cluster-uid is a required option")
	case region == "":
		log.Fatal("-region is a required option")
	case bucket == "":
		log.Fatal("-bucket is a required option")
	case queueURL == "":
		log.Fatal("-queue is a required option")
	case apiName == "":
		log.Fatal("-apiName is a required option")
	case jobID == "":
		log.Fatal("-jobID is a required option")
	}

	envConfig := enqueuer.EnvConfig{
		ClusterUID: clusterUID,
		Region:     region,
		Version:    version,
		Bucket:     bucket,
		APIName:    apiName,
		JobID:      jobID,
	}

	eqr, err := enqueuer.NewEnqueuer(envConfig, queueURL, log)
	if err != nil {
		log.Fatal("failed to create enqueuer", zap.Error(err))
	}

	totalBatches, err := eqr.Enqueue()
	if err != nil {
		log.Fatal("failed to enqueue batches", zap.Error(err))
	}

	if err = eqr.UploadBatchCount(totalBatches); err != nil {
		log.Fatal("failed to upload batch count", zap.Error(err))
	}

	log.Info("done enqueuing batches", zap.Int("batchCount", totalBatches))
}
