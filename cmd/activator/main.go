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
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/cortexlabs/cortex/pkg/activator"
	"github.com/cortexlabs/cortex/pkg/autoscaler"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/logging"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"go.uber.org/zap"
	istioinformers "istio.io/client-go/pkg/informers/externalversions"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kinformers "k8s.io/client-go/informers"
)

func main() {
	var (
		port          int
		adminPort     int
		inCluster     bool
		autoscalerURL string
		namespace     string
	)

	flag.IntVar(&port, "port", 8000, "port where the activator server will be exposed")
	flag.IntVar(&adminPort, "admin-port", 15000, "port where the admin server will be exposed")
	flag.BoolVar(&inCluster, "in-cluster", false, "use when autoscaler runs in-cluster")
	flag.StringVar(&autoscalerURL, "autoscaler-url", "", "the URL for the cortex autoscaler endpoint")
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
	case autoscalerURL == "":
		log.Fatal("--autoscaler-url is a required option")
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
			"image_type": "activator",
		},
		Environment: "operator",
		LogErrors:   true,
		BackoffMode: telemetry.BackoffDuplicateMessages,
	})
	if err != nil {
		log.Fatalw("failed to initialize telemetry", zap.Error(err))
	}
	defer telemetry.Close()

	k8sClient, err := k8s.New(namespace, inCluster, nil, runtime.NewScheme())
	if err != nil {
		exit(log, err, "failed to initialize kubernetes client")
	}

	istioClient := k8sClient.IstioClientSet()
	kubeClient := k8sClient.ClientSet()
	autoscalerClient := autoscaler.NewClient(autoscalerURL)

	prometheusStatsReporter := activator.NewPrometheusStatsReporter()

	istioInformerFactory := istioinformers.NewSharedInformerFactoryWithOptions(
		istioClient, 10*time.Second, // TODO: check how much makes sense
		istioinformers.WithNamespace(namespace),
		istioinformers.WithTweakListOptions(informerFilter),
	)
	virtualServiceInformer := istioInformerFactory.Networking().V1beta1().VirtualServices().Informer()
	virtualServiceClient := istioClient.NetworkingV1beta1().VirtualServices(namespace)

	kubeInformerFactory := kinformers.NewSharedInformerFactoryWithOptions(
		kubeClient, 2*time.Second, // TODO: check how much makes sense
		kinformers.WithNamespace(namespace),
		kinformers.WithTweakListOptions(informerFilter),
	)
	deploymentInformer := kubeInformerFactory.Apps().V1().Deployments().Informer()

	act := activator.New(
		virtualServiceClient,
		deploymentInformer,
		virtualServiceInformer,
		autoscalerClient,
		prometheusStatsReporter,
		log,
	)

	handler := activator.NewHandler(act, log)

	adminHandler := http.NewServeMux()
	adminHandler.Handle("/metrics", prometheusStatsReporter)

	servers := map[string]*http.Server{
		"activator": {
			Addr:    ":" + strconv.Itoa(port),
			Handler: handler,
		},
		"admin": {
			Addr:    ":" + strconv.Itoa(adminPort),
			Handler: adminHandler,
		},
	}

	stopCh := make(chan struct{})
	go virtualServiceInformer.Run(stopCh)
	go deploymentInformer.Run(stopCh)
	defer func() {
		stopCh <- struct{}{}
	}()

	errCh := make(chan error)
	for name, server := range servers {
		go func(name string, server *http.Server) {
			log.Infof("Starting %s server on %s", name, server.Addr)
			errCh <- server.ListenAndServe()
		}(name, server)
	}

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)

	select {
	case err = <-errCh:
		exit(log, err, "failed to start activator server")
	case <-sigint:
		log.Info("Received INT or TERM signal, handling a graceful shutdown...")

		for name, server := range servers {
			log.Infof("Shutting down %s server", name)
			if err = server.Shutdown(context.Background()); err != nil {
				// Error from closing listeners, or context timeout:
				log.Warnw("HTTP server Shutdown Error", zap.Error(err))
				telemetry.Error(errors.Wrap(err, "HTTP server Shutdown Error"))
			}
		}
		log.Info("Shutdown complete, exiting...")
	}
}

func informerFilter(listOptions *kmeta.ListOptions) {
	listOptions.LabelSelector = kmeta.FormatLabelSelector(&kmeta.LabelSelector{
		MatchLabels: map[string]string{
			"apiKind": userconfig.RealtimeAPIKind.String(),
		},
		MatchExpressions: []kmeta.LabelSelectorRequirement{
			{
				Key:      "apiName",
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
