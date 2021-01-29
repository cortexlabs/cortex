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
		fullClusterConfig := &clusterconfig.InternalConfig{
			OperatorMetadata: *OperatorMetadata,
		}

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

		fullClusterConfig.OperatorID = hashedAccountID
		fullClusterConfig.ClusterID = hash.String(fullClusterConfig.ClusterName + *fullClusterConfig.Region + hashedAccountID)
		Cluster = &fullClusterConfig.BaseConfig
		err = AWS.CreateDashboard(fullClusterConfig.ClusterName, consts.DashboardTitle)
		if err != nil {
			return err
		}
		exists, err := AWS.DoesBucketExist(fullClusterConfig.Bucket)
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
		gcpFullClusterConfig := &clusterconfig.InternalGCPConfig{
			OperatorMetadata: *OperatorMetadata,
		}

		bytes, _ := files.ReadFileBytes(clusterConfigPath)
		debug.Pp(string(bytes))

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

		fmt.Println("before NewFromEnvCheckProjectID")
		GCP, err = gcp.NewFromEnvCheckProjectID(*gcpFullClusterConfig.Project)
		if err != nil {
			return err
		}

		gcpFullClusterConfig.OperatorID = GCP.HashedProjectID
		gcpFullClusterConfig.ClusterID = hash.String(gcpFullClusterConfig.ClusterName + *gcpFullClusterConfig.Project + *gcpFullClusterConfig.Zone)

		debug.Pp(gcpFullClusterConfig)

		if gcpFullClusterConfig.Bucket == "" {
			fmt.Println("before CreateBucket")
			gcpFullClusterConfig.Bucket = clusterconfig.GCPBucketName(gcpFullClusterConfig.ClusterName, *gcpFullClusterConfig.Project, *gcpFullClusterConfig.Zone)
			err := GCP.CreateBucket(gcpFullClusterConfig.Bucket, gcp.ZoneToRegion(*gcpFullClusterConfig.Zone), true)
			if err != nil {
				return err
			}
		}

		// If the bucket is specified double check that it exists and the operator has access to it
		fmt.Println("")
		exists, err := GCP.DoesBucketExist(gcpFullClusterConfig.Bucket, gcp.ZoneToRegion(*gcpFullClusterConfig.Zone))
		if err != nil {
			return err
		}
		if !exists {
			return errors.ErrorUnexpected("the specified bucket either does not exist", gcpFullClusterConfig.Bucket)
		}

		GCPCluster = &gcpFullClusterConfig.GCPBaseConfig
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
