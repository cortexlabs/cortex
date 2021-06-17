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

	"github.com/cortexlabs/cortex/pkg/autoscaler"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/logging"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"github.com/gorilla/mux"
	promapi "github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"go.uber.org/zap"
	kcore "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func main() {
	var (
		port          int
		inCluster     bool
		prometheusURL string
	)

	flag.IntVar(&port, "port", 8000, "port where the autoscaler server will be exposed")
	flag.BoolVar(&inCluster, "in-cluster", false, "use when autoscaler runs in-cluster")
	flag.StringVar(&prometheusURL, "prometheus-url", os.Getenv("CORTEX_PROMETHEUS_URL"), "prometheus url (can be set through the CORTEX_PROMETHEUS_URL env variable)")

	log := logging.GetLogger()
	defer func() {
		_ = log.Sync()
	}()

	if prometheusURL == "" {
		log.Fatal("--prometheus-url is a required option")
	}

	k8sClient, err := k8s.New(kcore.NamespaceAll, inCluster, nil, runtime.NewScheme())
	if err != nil {
		log.Fatal("failed to initialize kubernetes client")
	}

	promClient, err := promapi.NewClient(
		promapi.Config{
			Address: prometheusURL,
		},
	)
	if err != nil {
		log.Fatal("failed to initialize prometheus client")
	}

	promAPIClient := promv1.NewAPI(promClient)

	realtimeScaler := autoscaler.NewRealtimeScaler(k8sClient, promAPIClient)
	asyncScaler := autoscaler.NewAsyncScaler(k8sClient, promAPIClient)

	autoScaler := autoscaler.New(log)
	autoScaler.AddScaler(realtimeScaler, userconfig.RealtimeAPIKind)
	autoScaler.AddScaler(asyncScaler, userconfig.AsyncAPIKind)

	handler := autoscaler.NewHandler(autoScaler)
	router := mux.NewRouter()
	router.HandleFunc("/add", handler.AddAPI).Methods(http.MethodPost)
	router.HandleFunc("/awake", handler.Awake).Methods(http.MethodPost)

	server := &http.Server{
		Addr:    ":" + strconv.Itoa(port),
		Handler: router,
	}

	errCh := make(chan error)
	go func() {
		log.Infof("Starting autoscaler server on %s", server.Addr)
		errCh <- server.ListenAndServe()
	}()

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt)

	select {
	case err = <-errCh:
		log.Fatalw("failed to start autoscaler server", zap.Error(err))
	case <-sigint:
		// We received an interrupt signal, shut down.
		log.Info("Received TERM signal, handling a graceful shutdown...")
		log.Info("Shutting down server")
		if err = server.Shutdown(context.Background()); err != nil {
			// Error from closing listeners, or context timeout:
			log.Warnw("HTTP server Shutdown Error", zap.Error(err))
		}
		log.Info("Shutdown complete, exiting...")
	}
}
