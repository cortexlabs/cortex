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

package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/cortexlabs/cortex/pkg/consts"
	batch "github.com/cortexlabs/cortex/pkg/crds/apis/batch/v1alpha1"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/hash"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
	promapi "github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
)

var (
	OperatorMetadata *clusterconfig.OperatorMetadata

	ClusterConfig *clusterconfig.Config

	AWS             *aws.Client
	K8s             *k8s.Client
	K8sIstio        *k8s.Client
	K8sAllNamspaces *k8s.Client
	MetricsClient   *statsd.Client
	Prometheus      promv1.API
	scheme          = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(batch.AddToScheme(scheme))
}

func InitConfigs(clusterConfig *clusterconfig.Config, operatorMetadata *clusterconfig.OperatorMetadata) {
	ClusterConfig = clusterConfig
	OperatorMetadata = operatorMetadata
}

func getClusterConfigFromConfigMap() (clusterconfig.Config, error) {
	configMapData, _, err := K8s.GetConfigMapData("cluster-config")
	if err != nil {
		return clusterconfig.Config{}, err
	}
	clusterConfig := clusterconfig.Config{}
	err = cr.ParseYAMLBytes(&clusterConfig, clusterconfig.FullConfigValidation, []byte(configMapData["cluster.yaml"]))
	if err != nil {
		return clusterconfig.Config{}, err
	}

	return clusterConfig, nil
}

func Init() error {
	var err error

	clusterConfigPath := os.Getenv("CORTEX_CLUSTER_CONFIG_PATH")
	if clusterConfigPath == "" {
		clusterConfigPath = consts.DefaultInClusterConfigPath
	}

	clusterConfig, err := clusterconfig.NewForFile(clusterConfigPath)
	if err != nil {
		return err
	}

	ClusterConfig = clusterConfig

	AWS, err = aws.NewForRegion(clusterConfig.Region)
	if err != nil {
		return err
	}

	accountID, hashedAccountID, err := AWS.CheckCredentials()
	if err != nil {
		return err
	}

	clusterConfig.AccountID = accountID

	OperatorMetadata = &clusterconfig.OperatorMetadata{
		APIVersion:          consts.CortexVersion,
		OperatorID:          hashedAccountID,
		ClusterID:           hash.String(clusterConfig.ClusterName + clusterConfig.Region + hashedAccountID),
		IsOperatorInCluster: strings.ToLower(os.Getenv("CORTEX_OPERATOR_IN_CLUSTER")) != "false",
	}

	if K8s, err = k8s.New(consts.DefaultNamespace, OperatorMetadata.IsOperatorInCluster, nil, scheme); err != nil {
		return err
	}

	if K8sIstio, err = k8s.New(consts.IstioNamespace, OperatorMetadata.IsOperatorInCluster, nil, scheme); err != nil {
		return err
	}

	if !OperatorMetadata.IsOperatorInCluster {
		cc, err := getClusterConfigFromConfigMap()
		if err != nil {
			return err
		}
		clusterConfig.Bucket = cc.Bucket
		clusterConfig.ClusterUID = cc.ClusterUID
	}

	exists, err := AWS.DoesBucketExist(clusterConfig.Bucket)
	if err != nil {
		return err
	}
	if !exists {
		return errors.ErrorUnexpected("the specified bucket does not exist", clusterConfig.Bucket)
	}

	err = telemetry.Init(telemetry.Config{
		Enabled: clusterConfig.Telemetry,
		UserID:  OperatorMetadata.OperatorID,
		Properties: map[string]string{
			"cluster_id":  OperatorMetadata.ClusterID,
			"operator_id": OperatorMetadata.OperatorID,
		},
		Environment: "operator",
		LogErrors:   true,
		BackoffMode: telemetry.BackoffDuplicateMessages,
	})
	if err != nil {
		fmt.Println(errors.Message(err))
	}

	prometheusURL := os.Getenv("CORTEX_PROMETHEUS_URL")
	if len(prometheusURL) == 0 {
		prometheusURL = fmt.Sprintf("http://prometheus.%s:9090", consts.PrometheusNamespace)
	}

	promClient, err := promapi.NewClient(promapi.Config{
		Address: prometheusURL,
	})
	if err != nil {
		return err
	}

	Prometheus = promv1.NewAPI(promClient)
	if K8sAllNamspaces, err = k8s.New("", OperatorMetadata.IsOperatorInCluster, nil, scheme); err != nil {
		return err
	}

	if OperatorMetadata.IsOperatorInCluster {
		MetricsClient, err = statsd.New(fmt.Sprintf("prometheus-statsd-exporter.%s:9125", consts.PrometheusNamespace))
		if err != nil {
			return errors.Wrap(errors.WithStack(err), "unable to initialize metrics client")
		}
	}

	return nil
}
