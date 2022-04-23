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
	"net/http"
	"os"
	"strings"

	gateway "github.com/cortexlabs/cortex/pkg/async-gateway"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/logging"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

const (
	_defaultPort = "8080"
)

// usage: ./gateway -bucket <bucket> -region <region> -port <port>
func main() {
	log := logging.GetLogger()
	defer func() {
		_ = log.Sync()
	}()

	var (
		bucket     = flag.String("bucket", "", "bucket")
		clusterUID = flag.String("cluster-uid", "", "cluster uid")
		port       = flag.String("port", _defaultPort, "port on which the gateway server runs on")
	)
	flag.Parse()

	switch {
	case *bucket == "":
		log.Fatal("missing required option: -bucket")
	case *clusterUID == "":
		log.Fatal("missing required option: -cluster-uid")
	}

	awsClient, err := aws.New()
	if err != nil {
		exit(log, err)
	}

	_, userID, err := awsClient.CheckCredentials()
	if err != nil {
		exit(log, err)
	}

	telemetryEnabled := strings.ToLower(os.Getenv("CORTEX_TELEMETRY_DISABLE")) != "true"
	err = telemetry.Init(telemetry.Config{
		Enabled: telemetryEnabled,
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
	s3Storage := gateway.NewS3(sess, *bucket)

	svc := gateway.NewService(*clusterUID, s3Storage, log, *sess)
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
