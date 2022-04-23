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

package batchcontrollers_test

import (
	// "os"
	"path/filepath"
	// "testing"

	// "github.com/cortexlabs/cortex/pkg/config"
	// "github.com/cortexlabs/cortex/pkg/consts"
	// batch "github.com/cortexlabs/cortex/pkg/crds/apis/batch/v1alpha1"
	// batchcontrollers "github.com/cortexlabs/cortex/pkg/crds/controllers/batch"
	awslib "github.com/cortexlabs/cortex/pkg/lib/aws"
	// "github.com/cortexlabs/cortex/pkg/lib/hash"
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
	// "github.com/cortexlabs/cortex/pkg/types/metrics"
	// . "github.com/onsi/ginkgo"
	// . "github.com/onsi/gomega"
	// "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	// ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	// "sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	// logf "sigs.k8s.io/controller-runtime/pkg/log"
	// "sigs.k8s.io/controller-runtime/pkg/log/zap"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var projectRoot = filepath.Join("..", "..", "..", "..")
var devClusterConfigPath = filepath.Join(projectRoot, "dev", "config", "cluster.yaml")

var cfg *rest.Config
var k8sClient client.Client
var awsClient *awslib.Client
var clusterConfig *clusterconfig.Config
var testEnv *envtest.Environment

// func TestAPIs(t *testing.T) {
// 	RegisterFailHandler(Fail)

// 	RunSpecsWithDefaultAndCustomReporters(t,
// 		"Controller Suite",
// 		[]Reporter{printer.NewlineReporter{}})
// }

// var _ = BeforeSuite(func(done Done) {
// 	logf.SetLogger(zap.New(zap.UseDevMode(true), zap.WriteTo(GinkgoWriter)))

// 	crdDirectoryPath := filepath.Join("..", "..", "config", "crd", "bases")
// 	Expect(crdDirectoryPath).To(BeADirectory())

// 	By("bootstrapping test environment")
// 	testEnv = &envtest.Environment{
// 		CRDDirectoryPaths: []string{crdDirectoryPath},
// 	}

// 	var err error
// 	cfg, err = testEnv.Start()
// 	Expect(err).ToNot(HaveOccurred())
// 	Expect(cfg).ToNot(BeNil())

// 	err = batch.AddToScheme(scheme.Scheme)
// 	Expect(err).NotTo(HaveOccurred())

// 	// +kubebuilder:scaffold:scheme

// 	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
// 	Expect(err).ToNot(HaveOccurred())
// 	Expect(k8sClient).ToNot(BeNil())

// 	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
// 		Scheme: scheme.Scheme,
// 	})
// 	Expect(err).ToNot(HaveOccurred())

// 	clusterConfigPath := os.Getenv("CORTEX_TEST_CLUSTER_CONFIG")
// 	if clusterConfigPath == "" {
// 		clusterConfigPath = devClusterConfigPath
// 	}

// 	clusterConfig, err = clusterconfig.NewForFile(clusterConfigPath)
// 	Expect(err).ToNot(HaveOccurred(),
// 		"error during cluster config creation (custom cluster "+
// 			"config paths can be set with the CORTEX_TEST_CLUSTER_CONFIG env variable)",
// 	)

// 	awsClient, err = awslib.NewForRegion(clusterConfig.Region)
// 	Expect(err).ToNot(HaveOccurred())

// 	accountID, hashedAccountID, err := awsClient.CheckCredentials()
// 	Expect(err).ToNot(HaveOccurred())

// 	clusterConfig.AccountID = accountID
// 	clusterConfig.Bucket = clusterconfig.BucketName(accountID, clusterConfig.ClusterName, clusterConfig.Region)

// 	operatorMetadata := &clusterconfig.OperatorMetadata{
// 		APIVersion:          consts.CortexVersion,
// 		OperatorID:          hashedAccountID,
// 		ClusterID:           hash.String(clusterConfig.ClusterName + clusterConfig.Region + hashedAccountID),
// 		IsOperatorInCluster: false,
// 	}

// 	// initialize some of the global values for the k8s helpers
// 	config.InitConfigs(clusterConfig, operatorMetadata)

// 	// mock certain methods of the reconciler
// 	reconcilerConfig := batchcontrollers.BatchJobReconcilerConfig{
// 		GetTotalBatchCount: func(r *batchcontrollers.BatchJobReconciler, batchJob batch.BatchJob) (int, error) {
// 			return 1, nil
// 		},
// 		GetMetrics: func(r *batchcontrollers.BatchJobReconciler, batchJob batch.BatchJob) (metrics.BatchMetrics, error) {
// 			return metrics.BatchMetrics{Succeeded: 1}, nil
// 		},
// 		SaveJobMetrics: func(r *batchcontrollers.BatchJobReconciler, batchJob batch.BatchJob) error {
// 			return nil
// 		},
// 		SaveJobStatus: func(r *batchcontrollers.BatchJobReconciler, batchJob batch.BatchJob) error {
// 			return nil
// 		},
// 	}

// 	err = (&batchcontrollers.BatchJobReconciler{
// 		Client:        k8sManager.GetClient(),
// 		Config:        reconcilerConfig,
// 		Log:           ctrl.Log.WithName("controllers").WithName("BatchJob"),
// 		ClusterConfig: clusterConfig,
// 		AWS:           awsClient,
// 		Scheme:        k8sManager.GetScheme(),
// 	}).SetupWithManager(k8sManager)
// 	Expect(err).ToNot(HaveOccurred())

// 	go func() {
// 		defer GinkgoRecover()
// 		err = k8sManager.Start(ctrl.SetupSignalHandler())
// 		Expect(err).ToNot(HaveOccurred())
// 	}()

// 	k8sClient = k8sManager.GetClient()
// 	Expect(k8sClient).ToNot(BeNil())

// 	close(done)
// }, 60)

// var _ = AfterSuite(func() {
// 	By("tearing down the test environment")
// 	err := testEnv.Stop()
// 	Expect(err).ToNot(HaveOccurred())
// })
