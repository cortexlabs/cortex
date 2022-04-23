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
	"fmt"
	"os"
	"strings"

	"github.com/cortexlabs/cortex/pkg/config"
	"github.com/cortexlabs/cortex/pkg/consts"
	awslib "github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/hash"
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
	promapi "github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	batch "github.com/cortexlabs/cortex/pkg/crds/apis/batch/v1alpha1"
	batchcontrollers "github.com/cortexlabs/cortex/pkg/crds/controllers/batch"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(batch.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var (
		metricsAddr          string
		enableLeaderElection bool
		probeAddr            string
		clusterConfigPath    string
		prometheusURL        string // defaults to http://prometheus.<namespace>:9090
		inCluster            = strings.ToLower(os.Getenv("CORTEX_OPERATOR_IN_CLUSTER")) == "true"
		useDevMode           = strings.ToLower(os.Getenv("CORTEX_DISABLE_JSON_LOGGING")) == "true"
	)
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&clusterConfigPath, "config", os.Getenv("CORTEX_CLUSTER_CONFIG_PATH"),
		"The path to the cluster config yaml file. "+
			"Can be set with the CORTEX_CLUSTER_CONFIG_PATH env variable. [Required]",
	)
	flag.StringVar(&prometheusURL, "prometheus-url", os.Getenv("CORTEX_PROMETHEUS_URL"),
		"Prometheus server URL",
	)

	opts := zap.Options{
		Development: useDevMode,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	switch {
	case clusterConfigPath == "":
		setupLog.Error(nil, "-config is a required flag")
		os.Exit(1)
	}

	clusterConfig, err := clusterconfig.NewForFile(clusterConfigPath)
	if err != nil {
		setupLog.Error(err, "failed to initialize cluster config")
		os.Exit(1)
	}

	if prometheusURL == "" {
		prometheusURL = fmt.Sprintf("http://prometheus.%s:9090", consts.PrometheusNamespace)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "7cc92962.cortex.dev",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	awsClient, err := awslib.NewForRegion(clusterConfig.Region)
	if err != nil {
		setupLog.Error(err, "failed to create AWS client")
		os.Exit(1)
	}

	accountID, hashedAccountID, err := awsClient.CheckCredentials()
	if err != nil {
		setupLog.Error(err, "failed to check AWS credentials")
		os.Exit(1)
	}

	clusterConfig.AccountID = accountID

	operatorMetadata := &clusterconfig.OperatorMetadata{
		APIVersion:          consts.CortexVersion,
		OperatorID:          hashedAccountID,
		ClusterID:           hash.String(clusterConfig.ClusterName + clusterConfig.Region + hashedAccountID),
		IsOperatorInCluster: inCluster,
	}

	promClient, err := promapi.NewClient(promapi.Config{Address: prometheusURL})
	if err != nil {
		setupLog.Error(err, "failed to initialize prometheus client")
		os.Exit(1)
	}

	// initialize some of the global values for the k8s helpers
	config.InitConfigs(clusterConfig, operatorMetadata)

	if err = (&batchcontrollers.BatchJobReconciler{
		Client:        mgr.GetClient(),
		Config:        batchcontrollers.BatchJobReconcilerConfig{}.ApplyDefaults(),
		Log:           ctrl.Log.WithName("controllers").WithName("BatchJob"),
		ClusterConfig: clusterConfig,
		AWS:           awsClient,
		Prometheus:    promv1.NewAPI(promClient),
		Scheme:        mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "BatchJob")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
