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
	"path"
	"strconv"
	"time"

	"github.com/cortexlabs/cortex/pkg/activator"
	"github.com/cortexlabs/cortex/pkg/lib/logging"
	"go.uber.org/zap"
	istioclient "istio.io/client-go/pkg/clientset/versioned"
	istioinformers "istio.io/client-go/pkg/informers/externalversions"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclientcmd "k8s.io/client-go/tools/clientcmd"
	kclienthomedir "k8s.io/client-go/util/homedir"
)

func main() {
	var (
		port int
	)

	flag.IntVar(&port, "port", 8000, "port where the activator server will be exposed")

	log := logging.GetLogger()
	defer func() {
		_ = log.Sync()
	}()

	kubeConfig := path.Join(kclienthomedir.HomeDir(), ".kube", "config")
	restConfig, err := kclientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		log.Fatal(err)
	}

	istioClient, err := istioclient.NewForConfig(restConfig)
	if err != nil {
		log.Fatal(err)
	}

	istioInformerFactory := istioinformers.NewSharedInformerFactory(istioClient, 10*time.Second) // TODO: check how much makes sense
	virtualServiceInformer := istioInformerFactory.Networking().V1beta1().VirtualServices().Informer()
	virtualServiceClient := istioClient.NetworkingV1beta1().VirtualServices(kmeta.NamespaceAll)

	act := activator.New(virtualServiceClient, virtualServiceInformer, log)
	handler := activator.NewHandler(act, log)
	server := &http.Server{
		Addr:    ":" + strconv.Itoa(port),
		Handler: handler,
	}

	stopCh := make(chan struct{})
	istioInformerFactory.Start(stopCh)
	defer func() {
		stopCh <- struct{}{}
	}()

	errCh := make(chan error)
	go func() {
		log.Infof("Starting activator server on %s", server.Addr)
		errCh <- server.ListenAndServe()
	}()

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt)

	select {
	case err = <-errCh:
		log.Fatalw("failed to start activator server", zap.Error(err))
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
