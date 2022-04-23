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
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/cortexlabs/cortex/pkg/autoscaler"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/logging"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"github.com/gorilla/mux"
	promapi "github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"go.uber.org/zap"
	istioclient "istio.io/client-go/pkg/clientset/versioned"
	istioinformers "istio.io/client-go/pkg/informers/externalversions"
	"k8s.io/apimachinery/pkg/api/meta"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"
)

func main() {
	var (
		port          int
		inCluster     bool
		prometheusURL string
		namespace     string
	)

	flag.IntVar(&port, "port", 8000, "port where the autoscaler server will be exposed")
	flag.BoolVar(&inCluster, "in-cluster", false, "use when autoscaler runs in-cluster")
	flag.StringVar(&prometheusURL, "prometheus-url", os.Getenv("CORTEX_PROMETHEUS_URL"),
		"prometheus url (can be set through the CORTEX_PROMETHEUS_URL env variable)",
	)
	flag.StringVar(&namespace, "namespace", os.Getenv("CORTEX_NAMESPACE"),
		"kubernetes namespace where the cortex APIs are deployed "+
			"(can be set through the CORTEX_NAMESPACE env variable)",
	)
	flag.Parse()

	log := logging.GetLogger()
	defer func() {
		_ = log.Sync()
	}()

	switch {
	case prometheusURL == "":
		log.Fatal("--prometheus-url is a required option")
	case namespace == "":
		log.Fatal("--namespace is a required option")
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
			"kind":       userconfig.RealtimeAPIKind.String(),
			"image_type": "autoscaler",
		},
		Environment: "operator",
		LogErrors:   true,
		BackoffMode: telemetry.BackoffDuplicateMessages,
	})
	if err != nil {
		log.Fatalw("failed to initialize telemetry", zap.Error(err))
	}
	defer telemetry.Close()

	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		exit(log, err, "failed to add k8s client-go-scheme to scheme")
	}

	k8sClient, err := k8s.New(namespace, inCluster, nil, scheme)
	if err != nil {
		exit(log, err, "failed to initialize kubernetes client")
	}

	//goland:noinspection GoNilness
	istioClient, err := istioclient.NewForConfig(k8sClient.RestConfig)
	if err != nil {
		exit(log, err, "failed to initialize istio client")
	}

	promClient, err := promapi.NewClient(
		promapi.Config{
			Address: prometheusURL,
		},
	)
	if err != nil {
		exit(log, err, "failed to initialize prometheus client")
	}

	promAPIClient := promv1.NewAPI(promClient)

	realtimeScaler := autoscaler.NewRealtimeScaler(k8sClient, promAPIClient, log)
	asyncScaler := autoscaler.NewAsyncScaler(k8sClient, promAPIClient)

	autoScaler := autoscaler.New(log)
	autoScaler.AddScaler(realtimeScaler, userconfig.RealtimeAPIKind)
	autoScaler.AddScaler(asyncScaler, userconfig.AsyncAPIKind)
	defer autoScaler.Stop()

	istioInformerFactory := istioinformers.NewSharedInformerFactoryWithOptions(
		istioClient, 10*time.Second, // TODO: check how much makes sense
		istioinformers.WithNamespace(namespace),
		istioinformers.WithTweakListOptions(informerFilter),
	)
	virtualServiceInformer := istioInformerFactory.Networking().V1beta1().VirtualServices().Informer()
	virtualServiceInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				resource, err := meta.Accessor(obj)
				if err != nil {
					log.Errorw("failed to access resource metadata", zap.Error(err))
					telemetry.Error(err)
					return
				}

				if resource.GetNamespace() != namespace {
					// filter out virtual services that are not in the cortex namespace
					return
				}

				api, err := apiResourceFromLabels(resource.GetLabels())
				if err != nil {
					// filter out non-cortex apis
					return
				}

				if err := autoScaler.AddAPI(api); err != nil {
					log.Errorw("failed to add API to autoscaler",
						zap.Error(err),
						zap.String("apiName", api.Name),
						zap.String("apiKind", api.Kind.String()),
					)
					telemetry.Error(err)
					return
				}
			},
			DeleteFunc: func(obj interface{}) {
				resource, err := meta.Accessor(obj)
				if err != nil {
					log.Errorw("failed to access resource metadata", zap.Error(err))
				}

				if resource.GetNamespace() != namespace {
					// filter out virtual services that are not in the cortex namespace
					return
				}

				api, err := apiResourceFromLabels(resource.GetLabels())
				if err != nil {
					// filter out non-cortex apis
					return
				}

				autoScaler.RemoveAPI(api)
			},
		},
	)

	handler := autoscaler.NewHandler(autoScaler)
	router := mux.NewRouter()
	router.HandleFunc("/awaken", handler.Awaken).Methods(http.MethodPost)
	router.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}).Methods(http.MethodGet)

	server := &http.Server{
		Addr:    ":" + strconv.Itoa(port),
		Handler: router,
	}

	stopCh := make(chan struct{})
	go virtualServiceInformer.Run(stopCh)
	defer func() { stopCh <- struct{}{} }()

	errCh := make(chan error)
	go func() {
		log.Infof("Starting autoscaler server on %s", server.Addr)
		errCh <- server.ListenAndServe()
	}()

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)

	select {
	case err = <-errCh:
		exit(log, err, "failed to start autoscaler server")
	case <-sigint:
		log.Info("Received INT or TERM signal, handling a graceful shutdown...")
		log.Info("Shutting down server")
		if err = server.Shutdown(context.Background()); err != nil {
			// Error from closing listeners, or context timeout:
			log.Warnw("HTTP server Shutdown Error", zap.Error(err))
		}
		log.Info("Shutdown complete, exiting...")
	}
}

func apiResourceFromLabels(labels map[string]string) (userconfig.Resource, error) {
	apiName, ok := labels["apiName"]
	if !ok {
		return userconfig.Resource{}, fmt.Errorf("apiName key does not exist")
	}

	apiKind, ok := labels["apiKind"]
	if !ok {
		return userconfig.Resource{}, fmt.Errorf("apiKind key does not exist")
	}

	return userconfig.Resource{
		Name: apiName,
		Kind: userconfig.KindFromString(apiKind),
	}, nil
}

func informerFilter(listOptions *kmeta.ListOptions) {
	listOptions.LabelSelector = kmeta.FormatLabelSelector(&kmeta.LabelSelector{
		MatchExpressions: []kmeta.LabelSelectorRequirement{
			{
				Key:      "apiName",
				Operator: kmeta.LabelSelectorOpExists,
			},
			{
				Key:      "apiKind",
				Operator: kmeta.LabelSelectorOpExists,
			},
		},
	})
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
