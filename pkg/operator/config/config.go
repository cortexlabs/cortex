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
	"github.com/cortexlabs/cortex/pkg/lib/files"

	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/debug"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/gcp"
	"github.com/cortexlabs/cortex/pkg/lib/hash"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/types"
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
)

const _clusterConfigPath = "/configs/cluster/cluster.yaml"

var (
	Provider         types.ProviderType
	OperatorMetadata *clusterconfig.OperatorMetadata

	Cluster              *clusterconfig.BaseConfig
	fullClusterConfig    *clusterconfig.InternalConfig
	GCPCluster           *clusterconfig.GCPBaseConfig
	gcpFullClusterConfig *clusterconfig.InternalGCPConfig

	AWS             *aws.Client
	GCP             *gcp.Client
	K8s             *k8s.Client
	K8sIstio        *k8s.Client
	K8sAllNamspaces *k8s.Client
)

func ManagedConfigOrNil() *clusterconfig.ManagedConfig {
	return &fullClusterConfig.ManagedConfig
}

func AWSInstanceMetadataOrNil() *aws.InstanceMetadata {
	return &fullClusterConfig.InstanceMetadata
}

func FullClusterConfig() *clusterconfig.InternalConfig {
	return fullClusterConfig
}

func GCPManagedConfigOrNil() *clusterconfig.GCPManagedConfig {
	return &gcpFullClusterConfig.GCPManagedConfig
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

	if Provider == types.AWSProviderType {
		OperatorMetadata = &clusterconfig.OperatorMetadata{
			APIVersion:          consts.CortexVersion,
			IsOperatorInCluster: strings.ToLower(os.Getenv("CORTEX_OPERATOR_IN_CLUSTER")) != "false",
		}
		cluster3 := &clusterconfig.InternalConfig{
			OperatorMetadata: *OperatorMetadata,
		}

		errs := cr.ParseYAMLFile(cluster3, clusterconfig.BaseConfigValidations(true), clusterConfigPath)
		if errors.HasError(errs) {
			return errors.FirstError(errs...)
		}

		if cluster3.IsManaged {
			errs := cr.ParseYAMLFile(cluster3, clusterconfig.ManagedConfigValidations(true), clusterConfigPath)
			if errors.HasError(errs) {
				return errors.FirstError(errs...)
			}
			cluster3.InstanceMetadata = aws.InstanceMetadatas[*cluster3.Region][*cluster3.InstanceType]
		}

		AWS, err = aws.NewFromEnv(*cluster3.Region)
		if err != nil {
			return err
		}

		_, hashedAccountID, err := AWS.CheckCredentials()
		if err != nil {
			return err
		}

		cluster3.OperatorID = hashedAccountID
		cluster3.ClusterID = hash.String(cluster3.ClusterName + *cluster3.Region + hashedAccountID)
		Cluster = &cluster3.BaseConfig
		fullClusterConfig = cluster3
		err = AWS.CreateDashboard(cluster3.ClusterName, consts.DashboardTitle)
		if err != nil {
			return err
		}
		exists, err := AWS.DoesBucketExist(cluster3.Bucket)
		if err != nil {
			return err
		}
		if !exists {
			return errors.ErrorUnexpected("the specified bucket either does not exist", cluster3.Bucket)
		}
		clusterNamespace = Cluster.Namespace
		istioNamespace = Cluster.IstioNamespace
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
		cluster3 := &clusterconfig.InternalGCPConfig{
			OperatorMetadata: *OperatorMetadata,
		}

		bytes, _ := files.ReadFileBytes(clusterConfigPath)
		debug.Pp(string(bytes))

		errs := cr.ParseYAMLFile(cluster3, clusterconfig.GCPBaseConfigValidations(true), clusterConfigPath)
		if errors.HasError(errs) {
			return errors.FirstError(errs...)
		}

		if cluster3.IsManaged {
			errs := cr.ParseYAMLFile(cluster3, clusterconfig.GCPManagedConfigValidations(true), clusterConfigPath)
			if errors.HasError(errs) {
				return errors.FirstError(errs...)
			}
		}

		fmt.Println("before NewFromEnvCheckProjectID")
		GCP, err = gcp.NewFromEnvCheckProjectID(*cluster3.Project)
		if err != nil {
			return err
		}

		cluster3.OperatorID = GCP.HashedProjectID
		cluster3.ClusterID = hash.String(cluster3.ClusterName + *cluster3.Project + *cluster3.Zone)

		debug.Pp(cluster3)

		if cluster3.Bucket == "" {
			fmt.Println("before CreateBucket")
			cluster3.Bucket = clusterconfig.GCPBucketName(cluster3.ClusterName, *cluster3.Project, *cluster3.Zone)
			err := GCP.CreateBucket(cluster3.Bucket, gcp.ZoneToRegion(*cluster3.Zone), true)
			if err != nil {
				return err
			}
		}

		// If the bucket is specified double check that it exists and the operator has access to it
		fmt.Println("")
		exists, err := GCP.DoesBucketExist(cluster3.Bucket, gcp.ZoneToRegion(*cluster3.Zone))
		if err != nil {
			return err
		}
		if !exists {
			return errors.ErrorUnexpected("the specified bucket either does not exist", cluster3.Bucket)
		}

		gcpFullClusterConfig = cluster3
		GCPCluster = &cluster3.GCPBaseConfig
		clusterNamespace = GCPCluster.Namespace
		istioNamespace = GCPCluster.IstioNamespace
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

	if K8s, err = k8s.New(clusterNamespace, IsOperatorInCluster(), nil); err != nil {
		return err
	}

	if K8sIstio, err = k8s.New(istioNamespace, IsOperatorInCluster(), nil); err != nil {
		return err
	}

	if K8sAllNamspaces, err = k8s.New("", IsOperatorInCluster(), nil); err != nil {
		return err
	}

	return nil
}
