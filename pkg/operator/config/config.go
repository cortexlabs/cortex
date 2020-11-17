/*
Copyright 2020 Cortex Labs, Inc.

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
	"github.com/cortexlabs/cortex/pkg/lib/debug"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/gcp"
	"github.com/cortexlabs/cortex/pkg/lib/hash"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/types"
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
	"gopkg.in/yaml.v2"
)

const _clusterConfigPath = "/configs/cluster/cluster.yaml"
const _clusterConfigBackupPath = "/configs/cluster-aws/cluster-aws.yaml"

var (
	Cluster *clusterconfig.InternalConfig
	AWS     *aws.Client
	GCP     *gcp.Client

	K8s             *k8s.Client
	K8sIstio        *k8s.Client
	K8sAllNamspaces *k8s.Client
	GCPCluster      *clusterconfig.InternalGCPConfig
	Provider        types.ProviderType
)

func Init() error {
	var err error

	Cluster = &clusterconfig.InternalConfig{
		APIVersion:        consts.CortexVersion,
		OperatorInCluster: strings.ToLower(os.Getenv("CORTEX_OPERATOR_IN_CLUSTER")) != "false",
	}

	clusterConfigPath := os.Getenv("CORTEX_CLUSTER_CONFIG_PATH")
	if clusterConfigPath == "" {
		clusterConfigPath = _clusterConfigPath
	}

	Provider, err = clusterconfig.GetClusterProviderType(clusterConfigPath)
	if err != nil {
		return err
	}

	hashedAccountID := ""

	awsClusterConfigPath := clusterConfigPath
	if Provider == types.GCPProviderType {
		gcpClusterconfigBytes, err := files.ReadFileBytes(clusterConfigPath)
		if err != nil {
			return err
		}

		gcpCluster := clusterconfig.GCPConfig{}

		err = yaml.Unmarshal(gcpClusterconfigBytes, &gcpCluster)
		if err != nil {
			return err
		}

		id := hash.String(gcpCluster.ClusterName + gcpCluster.Zone + gcpCluster.Project)

		GCPCluster = &clusterconfig.InternalGCPConfig{
			GCPConfig:         gcpCluster,
			ID:                id,
			APIVersion:        consts.CortexVersion,
			OperatorInCluster: strings.ToLower(os.Getenv("CORTEX_OPERATOR_IN_CLUSTER")) != "false",
		}

		awsClusterConfigPath = os.Getenv("CORTEX_AWS_CLUSTER_CONFIG_PATH")
		if awsClusterConfigPath == "" {
			awsClusterConfigPath = _clusterConfigBackupPath
		}

		GCP = &gcp.Client{}
	}

	errs := cr.ParseYAMLFile(Cluster, clusterconfig.Validation, awsClusterConfigPath)
	if errors.HasError(errs) {
		return errors.FirstError(errs...)
	}

	Cluster.InstanceMetadata = aws.InstanceMetadatas[*Cluster.Region][*Cluster.InstanceType]

	debug.Pp(os.Environ())

	AWS, err = aws.NewFromEnv(*Cluster.Region)
	if err != nil {
		return err
	}

	_, hashedAccountID, err = AWS.CheckCredentials()
	if err != nil {
		return err
	}
	Cluster.ID = hash.String(Cluster.ClusterName + *Cluster.Region + hashedAccountID)

	err = telemetry.Init(telemetry.Config{
		Enabled: Cluster.Telemetry,
		UserID:  hashedAccountID,
		Properties: map[string]string{
			"cluster_id":  Cluster.ID,
			"operator_id": hashedAccountID,
		},
		Environment: "operator",
		LogErrors:   true,
		BackoffMode: telemetry.BackoffDuplicateMessages,
	})
	if err != nil {
		fmt.Println(errors.Message(err))
	}

	debug.Pp(Cluster)
	debug.Pp(GCPCluster)

	// if Cluster.APIGatewaySetting == clusterconfig.PublicAPIGatewaySetting {
	// 	apiGateway, err := AWS.GetAPIGatewayByTag(clusterconfig.ClusterNameTag, Cluster.ClusterName)
	// 	if err != nil {
	// 		return err
	// 	} else if apiGateway == nil {
	// 		return ErrorNoAPIGateway()
	// 	}
	// 	Cluster.APIGateway = apiGateway

	// 	if Cluster.APILoadBalancerScheme == clusterconfig.InternalLoadBalancerScheme {
	// 		vpcLink, err := AWS.GetVPCLinkByTag(clusterconfig.ClusterNameTag, Cluster.ClusterName)
	// 		if err != nil {
	// 			return err
	// 		} else if vpcLink == nil {
	// 			return ErrorNoVPCLink()
	// 		}
	// 		Cluster.VPCLink = vpcLink

	// 		integration, err := AWS.GetVPCLinkIntegration(*Cluster.APIGateway.ApiId, *Cluster.VPCLink.VpcLinkId)
	// 		if err != nil {
	// 			return err
	// 		} else if integration == nil {
	// 			return ErrorNoVPCLinkIntegration()
	// 		}
	// 		Cluster.VPCLinkIntegration = integration
	// 	}
	// }

	if K8s, err = k8s.New("default", Cluster.OperatorInCluster); err != nil {
		return err
	}

	if K8sIstio, err = k8s.New("istio-system", Cluster.OperatorInCluster); err != nil {
		return err
	}

	if K8sAllNamspaces, err = k8s.New("", Cluster.OperatorInCluster); err != nil {
		return err
	}

	return nil
}
