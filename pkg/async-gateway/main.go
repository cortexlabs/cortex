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

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"github.com/cortexlabs/cortex/pkg/lib/aws"
	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/logging"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
)

const (
	_defaultPort = "8080"
)

var (
	gatewayLogger = logging.GetLogger()
)

func Error(err error, wrapStrs ...string) {
	for _, str := range wrapStrs {
		err = errors.Wrap(err, str)
	}

	if err != nil && !errors.IsNoTelemetry(err) {
		telemetry.Error(err)
	}

	if err != nil && !errors.IsNoPrint(err) {
		gatewayLogger.Error(err)
	}

	telemetry.Close()

	os.Exit(1)
}

// usage: ./gateway -bucket <bucket> -region <region> -port <port> -queue queue <apiName>
func main() {
	log := logging.GetLogger()
	defer func() {
		_ = log.Sync()
	}()

	var (
		port              = flag.String("port", _defaultPort, "port on which the gateway server runs on")
		queueURL          = flag.String("queue", "", "SQS queue URL")
		region            = flag.String("region", "", "AWS region")
		bucket            = flag.String("bucket", "", "AWS bucket")
		clusterName       = flag.String("cluster", "", "cluster name")
		clusterConfigPath = flag.String("cluster-config-path", "", "cluster config path")
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
	case *clusterConfigPath == "":
		log.Fatal("missing required option: -cluster-config-path")
	}

	apiName := flag.Arg(0)
	if apiName == "" {
		log.Fatal("apiName argument was not provided")
	}

	coreConfig := &clusterconfig.CoreConfig{}
	errs := cr.ParseYAMLFile(coreConfig, clusterconfig.CoreConfigValidations(true), *clusterConfigPath)
	if errors.HasError(errs) {
		Error(errors.FirstError(errs...))
	}

	aws, err := aws.NewForRegion(*region)
	if err != nil {
		Error(err)
	}

	_, userID, err := aws.CheckCredentials()
	if err != nil {
		Error(err)
	}

	err = telemetry.Init(telemetry.Config{
		Enabled: coreConfig.Telemetry,
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
		Error(err)
	}

	sess := aws.Session()
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
		Error(err)
	}
}
