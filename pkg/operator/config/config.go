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

package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/aws"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/hash"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
	promapi "github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

const (
	_clusterConfigPath = "/configs/cluster/cluster.yaml"
)

var (
	OperatorMetadata *clusterconfig.OperatorMetadata

	ClusterConfig     *clusterconfig.Config
	InstancesMetadata []aws.InstanceMetadata

	AWS             *aws.Client
	K8s             *k8s.Client
	K8sIstio        *k8s.Client
	K8sAllNamspaces *k8s.Client
	Prometheus      promv1.API
)

func Init() error {
	var err error
	var clusterNamespace string
	var istioNamespace string

	clusterConfigPath := os.Getenv("CORTEX_CLUSTER_CONFIG_PATH")
	if clusterConfigPath == "" {
		clusterConfigPath = _clusterConfigPath
	}

	clusterConfig, err := clusterconfig.NewForFile(clusterConfigPath)
	if err != nil {
		return err
	}

	ClusterConfig = clusterConfig

	for _, instanceType := range clusterConfig.GetAllInstanceTypes() {
		InstancesMetadata = append(InstancesMetadata, aws.InstanceMetadatas[clusterConfig.Region][instanceType])
	}

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

	clusterNamespace = clusterConfig.Namespace
	istioNamespace = clusterConfig.IstioNamespace

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

	if K8s, err = k8s.New(clusterNamespace, OperatorMetadata.IsOperatorInCluster, nil); err != nil {
		return err
	}

	if K8sIstio, err = k8s.New(istioNamespace, OperatorMetadata.IsOperatorInCluster, nil); err != nil {
		return err
	}

	prometheusURL := os.Getenv("CORTEX_PROMETHEUS_URL")
	if len(prometheusURL) == 0 {
		prometheusURL = fmt.Sprintf("http://prometheus.%s:9090", clusterNamespace)
	}

	promClient, err := promapi.NewClient(promapi.Config{
		Address: prometheusURL,
	})
	if err != nil {
		return err
	}

	Prometheus = promv1.NewAPI(promClient)

	if K8sAllNamspaces, err = k8s.New("", OperatorMetadata.IsOperatorInCluster, nil); err != nil {
		return err
	}

	return nil
}
