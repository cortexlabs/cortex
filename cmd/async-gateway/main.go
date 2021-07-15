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

	gateway "github.com/cortexlabs/cortex/pkg/async-gateway"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/logging"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

const (
	_defaultPort = "8080"
)

// usage: ./gateway -bucket <bucket> -region <region> -port <port> -queue queue <apiName>
func main() {
	log := logging.GetLogger()
	defer func() {
		_ = log.Sync()
	}()

	var (
		clusterConfigPath = flag.String("cluster-config", "", "cluster config path")
		port              = flag.String("port", _defaultPort, "port on which the gateway server runs on")
		queueURL          = flag.String("queue", "", "SQS queue URL")
	)
	flag.Parse()

	switch {
	case *queueURL == "":
		log.Fatal("missing required option: -queue")
	case *clusterConfigPath == "":
		log.Fatal("missing required option: -cluster-config")
	}

	apiName := flag.Arg(0)
	if apiName == "" {
		log.Fatal("apiName argument was not provided")
	}

	clusterConfig, err := clusterconfig.NewForFile(*clusterConfigPath)
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
			"kind":       userconfig.AsyncAPIKind.String(),
			"image_type": "async-gateway",
		},
		Environment: "api",
		LogErrors:   true,
		BackoffMode: telemetry.BackoffDuplicateMessages,
	})
	if err != nil {
		log.Fatalw("failed to initialize telemetry", zap.Error(err))
	}
	defer telemetry.Close()

	sess := awsClient.Session()
	s3Storage := gateway.NewS3(sess, clusterConfig.Bucket)
	sqsQueue := gateway.NewSQS(*queueURL, sess)

	svc := gateway.NewService(clusterConfig.ClusterUID, apiName, sqsQueue, s3Storage, log)
	ep := gateway.NewEndpoint(svc, log)

	router := mux.NewRouter()
	router.HandleFunc("/", ep.CreateWorkload).Methods("POST")
	router.HandleFunc(
		"/healthz",
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		},
	)
	router.HandleFunc("/{id}", ep.GetWorkload).Methods("GET")

	// inspired by our nginx config
	corsOptions := []handlers.CORSOption{
		handlers.AllowedOrigins([]string{"*"}),
		// custom headers are not supported currently, since "*" is not supported in AllowedHeaders(); here are some common ones:
		handlers.AllowedHeaders([]string{"Content-Type", "X-Requested-With", "User-Agent", "Accept", "Accept-Language", "Content-Language", "Origin"}),
		handlers.AllowedMethods([]string{"GET", "HEAD", "POST", "PUT", "OPTIONS"}),
		handlers.ExposedHeaders([]string{"Content-Length", "Content-Range"}),
		handlers.AllowCredentials(),
	}

	log.Info("Running on port " + *port)
	if err = http.ListenAndServe(":"+*port, handlers.CORS(corsOptions...)(router)); err != nil {
		exit(log, err)
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
