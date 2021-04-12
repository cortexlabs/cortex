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
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sts"
	batch "github.com/cortexlabs/cortex/operator/apis/batch/v1alpha1"
	controllers "github.com/cortexlabs/cortex/operator/controllers/batch"
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)

	_ = batch.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	var (
		metricsAddr          string
		enableLeaderElection bool
		clusterConfigPath    string
	)
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&clusterConfigPath, "config", os.Getenv("CORTEX_CLUSTER_CONFIG_PATH"),
		"The path to the cluster config yaml file. "+
			"Can be set with the CORTEX_CLUSTER_CONFIG_PATH env variable. [Required]",
	)
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

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

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "7cc92962.cortex.dev",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	sess, err := session.NewSessionWithOptions(session.Options{
		Config: aws.Config{
			Region: aws.String(clusterConfig.Region),
		},
		SharedConfigState: session.SharedConfigEnable,
	})
	if err != nil {
		setupLog.Error(err, "failed to create AWS session")
		os.Exit(1)
	}

	stsClient := sts.New(sess)
	response, err := stsClient.GetCallerIdentity(nil)
	if err != nil {
		setupLog.Error(err, "failed to retrieve AWS credentials")
		os.Exit(1)
	}
	clusterConfig.AccountID = *response.Account

	if err = (&controllers.BatchJobReconciler{
		Client:        mgr.GetClient(),
		Log:           ctrl.Log.WithName("controllers").WithName("BatchJob"),
		ClusterConfig: clusterConfig,
		SQS:           sqs.New(sess),
		Scheme:        mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "BatchJob")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err = mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
