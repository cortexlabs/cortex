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
	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/gcp"
	"github.com/cortexlabs/cortex/pkg/lib/hash"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/types"
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
)

const (
	_clusterConfigPath   = "/configs/cluster/cluster.yaml"
	DefaultPrometheusURL = "http://prometheus.default:9090"
)

var (
	Provider        types.ProviderType
	Cluster         *clusterconfig.InternalConfig
	GCPCluster      *clusterconfig.InternalGCPConfig
	AWS             *aws.Client
	GCP             *gcp.Client
	K8s             *k8s.Client
	K8sIstio        *k8s.Client
	K8sAllNamspaces *k8s.Client
)

func Init() error {
	var err error

	clusterConfigPath := os.Getenv("CORTEX_CLUSTER_CONFIG_PATH")
	if clusterConfigPath == "" {
		clusterConfigPath = _clusterConfigPath
	}

	Provider, err = clusterconfig.GetClusterProviderType(clusterConfigPath)
	if err != nil {
		return err
	}

	prometheusURL := os.Getenv("CORTEX_PROMETHEUS_URL")
	if len(prometheusURL) == 0 {
		prometheusURL = DefaultPrometheusURL
	}

	if Provider == types.AWSProviderType {
		Cluster = &clusterconfig.InternalConfig{
			APIVersion:          consts.CortexVersion,
			IsOperatorInCluster: strings.ToLower(os.Getenv("CORTEX_OPERATOR_IN_CLUSTER")) != "false",
			PrometheusURL:       prometheusURL,
		}

		errs := cr.ParseYAMLFile(Cluster, clusterconfig.Validation, clusterConfigPath)
		if errors.HasError(errs) {
			return errors.FirstError(errs...)
		}

		Cluster.InstanceMetadata = aws.InstanceMetadatas[*Cluster.Region][*Cluster.InstanceType]

		AWS, err = aws.NewFromEnv(*Cluster.Region)
		if err != nil {
			return err
		}

		_, hashedAccountID, err := AWS.CheckCredentials()
		if err != nil {
			return err
		}
		Cluster.OperatorID = hashedAccountID
		Cluster.ClusterID = hash.String(Cluster.ClusterName + *Cluster.Region + hashedAccountID)
	} else {
		AWS, err = aws.NewAnonymousClient()
		if err != nil {
			return err
		}
	}

	if Provider == types.GCPProviderType {
		GCPCluster = &clusterconfig.InternalGCPConfig{
			APIVersion:          consts.CortexVersion,
			IsOperatorInCluster: strings.ToLower(os.Getenv("CORTEX_OPERATOR_IN_CLUSTER")) != "false",
			PrometheusURL:       prometheusURL,
		}

		errs := cr.ParseYAMLFile(GCPCluster, clusterconfig.GCPValidation, clusterConfigPath)
		if errors.HasError(errs) {
			return errors.FirstError(errs...)
		}

		GCP, err = gcp.NewFromEnvCheckProjectID(*GCPCluster.Project)
		if err != nil {
			return err
		}

		GCPCluster.OperatorID = GCP.HashedProjectID
		GCPCluster.ClusterID = hash.String(GCPCluster.ClusterName + *GCPCluster.Project + *GCPCluster.Zone)

		GCPCluster.Bucket = clusterconfig.GCPBucketName(GCPCluster.ClusterName, *GCPCluster.Project, *GCPCluster.Zone)
	} else {
		GCP = gcp.NewAnonymousClient()
	}

	err = telemetry.Init(telemetry.Config{
		Enabled: Telemetry(),
		UserID:  OperatorID(),
		Properties: map[string]string{
			"cluster_id":  ClusterID(),
			"operator_id": OperatorID(),
		},
		Environment: "operator",
		LogErrors:   true,
		BackoffMode: telemetry.BackoffDuplicateMessages,
	})
	if err != nil {
		fmt.Println(errors.Message(err))
	}

	if K8s, err = k8s.New("default", IsOperatorInCluster(), nil); err != nil {
		return err
	}

	if K8sIstio, err = k8s.New("istio-system", IsOperatorInCluster(), nil); err != nil {
		return err
	}

	if K8sAllNamspaces, err = k8s.New("", IsOperatorInCluster(), nil); err != nil {
		return err
	}

	return nil
}
