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
	promapi "github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

const (
	_clusterConfigPath   = "/configs/cluster/cluster.yaml"
	DefaultPrometheusURL = "http://prometheus.default:9090"
)

var (
	Provider         types.ProviderType
	OperatorMetadata *clusterconfig.OperatorMetadata

	Cluster           *clusterconfig.BaseConfig     // CoreConfig *clusterconfig.CoreConfig
	fullClusterConfig *clusterconfig.InternalConfig // managedConfig *clusterconfig.managedConfig
	// instanceMetadata *clusterconfig.InstanceMetadata
	GCPCluster           *clusterconfig.GCPBaseConfig     // GCPCoreConfig *clusterconfig.GCPCoreConfig
	gcpFullClusterConfig *clusterconfig.InternalGCPConfig //

	AWS             *aws.Client
	GCP             *gcp.Client
	K8s             *k8s.Client
	K8sIstio        *k8s.Client
	K8sAllNamspaces *k8s.Client
	Prometheus      promv1.API
)

func ManagedConfigOrNil() *clusterconfig.ManagedConfig {
	if Cluster.IsManaged {
		return &fullClusterConfig.ManagedConfig
	}
	return nil
}

func AWSInstanceMetadataOrNil() *aws.InstanceMetadata {
	if Cluster.IsManaged {
		return &fullClusterConfig.InstanceMetadata
	}
	return nil
}

func FullClusterConfig() *clusterconfig.InternalConfig {
	return fullClusterConfig
}

func GCPManagedConfigOrNil() *clusterconfig.GCPManagedConfig {
	if GCPCluster.IsManaged {
		return &gcpFullClusterConfig.GCPManagedConfig
	}
	return nil
}

func GCPFullClusterConfig() *clusterconfig.InternalGCPConfig {
	return gcpFullClusterConfig
}

func Init() error {
	var err error
	var clusterNamespace string
	var istioNamespace string

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
		OperatorMetadata = &clusterconfig.OperatorMetadata{
			APIVersion:          consts.CortexVersion,
			IsOperatorInCluster: strings.ToLower(os.Getenv("CORTEX_OPERATOR_IN_CLUSTER")) != "false",
		}

		fullClusterConfig = &clusterconfig.InternalConfig{}

		errs := cr.ParseYAMLFile(fullClusterConfig, clusterconfig.BaseConfigValidations(true), clusterConfigPath)
		if errors.HasError(errs) {
			return errors.FirstError(errs...)
		}

		if fullClusterConfig.IsManaged {
			errs := cr.ParseYAMLFile(fullClusterConfig, clusterconfig.ManagedConfigValidations(true), clusterConfigPath)
			if errors.HasError(errs) {
				return errors.FirstError(errs...)
			}
			fullClusterConfig.InstanceMetadata = aws.InstanceMetadatas[*fullClusterConfig.Region][*fullClusterConfig.InstanceType]
		}

		AWS, err = aws.NewFromEnv(*fullClusterConfig.Region)
		if err != nil {
			return err
		}

		_, hashedAccountID, err := AWS.CheckCredentials()
		if err != nil {
			return err
		}

		OperatorMetadata.OperatorID = hashedAccountID
		OperatorMetadata.ClusterID = hash.String(fullClusterConfig.ClusterName + *fullClusterConfig.Region + hashedAccountID)
		fullClusterConfig.OperatorMetadata = *OperatorMetadata

		Cluster = &fullClusterConfig.BaseConfig
		clusterNamespace = Cluster.Namespace
		istioNamespace = Cluster.IstioNamespace

		err = AWS.CreateDashboard(fullClusterConfig.ClusterName, consts.DashboardTitle)
		if err != nil {
			return err
		}
		exists, err := AWS.DoesBucketExist(fullClusterConfig.Bucket)
		if err != nil {
			return err
		}
		if !exists {
			return errors.ErrorUnexpected("the specified bucket does not exist", fullClusterConfig.Bucket)
		}
	} else {
		AWS, err = aws.NewAnonymousClient()
		if err != nil {
			return err
		}
	}

	if Provider == types.GCPProviderType {
		OperatorMetadata = &clusterconfig.OperatorMetadata{
			APIVersion:          consts.CortexVersion,
			IsOperatorInCluster: strings.ToLower(os.Getenv("CORTEX_OPERATOR_IN_CLUSTER")) != "false",
		}
		gcpFullClusterConfig = &clusterconfig.InternalGCPConfig{}

		errs := cr.ParseYAMLFile(gcpFullClusterConfig, clusterconfig.GCPBaseConfigValidations(true), clusterConfigPath)
		if errors.HasError(errs) {
			return errors.FirstError(errs...)
		}

		if gcpFullClusterConfig.IsManaged {
			errs := cr.ParseYAMLFile(gcpFullClusterConfig, clusterconfig.GCPManagedConfigValidations(true), clusterConfigPath)
			if errors.HasError(errs) {
				return errors.FirstError(errs...)
			}
		}

		GCP, err = gcp.NewFromEnvCheckProjectID(*gcpFullClusterConfig.Project)
		if err != nil {
			return err
		}

		OperatorMetadata.OperatorID = GCP.HashedProjectID
		OperatorMetadata.ClusterID = hash.String(gcpFullClusterConfig.ClusterName + *gcpFullClusterConfig.Project + *gcpFullClusterConfig.Zone)
		gcpFullClusterConfig.OperatorMetadata = *OperatorMetadata

		GCPCluster = &gcpFullClusterConfig.GCPBaseConfig
		clusterNamespace = GCPCluster.Namespace
		istioNamespace = GCPCluster.IstioNamespace

		// If the bucket is specified double check that it exists and the operator has access to it
		exists, err := GCP.DoesBucketExist(gcpFullClusterConfig.Bucket, gcp.ZoneToRegion(*gcpFullClusterConfig.Zone))
		if err != nil {
			return err
		}
		if !exists {
			return errors.ErrorUnexpected("the specified bucket does not exist", gcpFullClusterConfig.Bucket)
		}
	} else {
		GCP = gcp.NewAnonymousClient()
	}

	err = telemetry.Init(telemetry.Config{
		Enabled: Telemetry(),
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
