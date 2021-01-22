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
	"github.com/cortexlabs/cortex/pkg/lib/debug"

	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
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

var (
	Provider        types.ProviderType
	Cluster         *clusterconfig.InternalConfig2
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

	if Provider == types.AWSProviderType {
		// cluster := &clusterconfig.InternalConfig{
		// 	APIVersion:          consts.CortexVersion,
		// 	IsOperatorInCluster: strings.ToLower(os.Getenv("CORTEX_OPERATOR_IN_CLUSTER")) != "false",
		// }

		// errs := cr.ParseYAMLFile(cluster, clusterconfig.Validation, clusterConfigPath)
		// if errors.HasError(errs) {
		// 	return errors.FirstError(errs...)
		// }

		fileBytes, err := files.ReadFileBytes(clusterConfigPath)
		if err != nil {
			return err
		}
		fmt.Println(string(fileBytes))

		baseConfig := clusterconfig.BaseConfig{}

		err = yaml.Unmarshal(fileBytes, &baseConfig)
		if err != nil {
			return err
		}

		debug.Pp(baseConfig)

		// Cluster.InstanceMetadata = aws.InstanceMetadatas[*Cluster.Region][*Cluster.InstanceType] // TODO

		AWS, err = aws.NewFromEnv(*baseConfig.Region)
		if err != nil {
			return err
		}

		_, hashedAccountID, err := AWS.CheckCredentials()
		if err != nil {
			return err
		}
		internalConfig2 := clusterconfig.InternalConfig2{BaseConfig: baseConfig}
		internalConfig2.APIVersion = consts.CortexVersion
		internalConfig2.IsOperatorInCluster = strings.ToLower(os.Getenv("CORTEX_OPERATOR_IN_CLUSTER")) != "false"
		internalConfig2.OperatorID = hashedAccountID
		internalConfig2.ClusterID = hash.String(baseConfig.ClusterName + *baseConfig.Region + hashedAccountID)
		Cluster = &internalConfig2
		// Cluster = &clusterconfig.InternalConfig2{
		// 	BaseConfig: clusterconfig.BaseConfig{
		// 		ClusterName:         cluster.ClusterName,
		// 		Region:              cluster.Region,
		// 		Bucket:              cluster.Bucket,
		// 		Provider:            cluster.Provider,
		// 		Telemetry:           cluster.Telemetry,
		// 		ImageDownloader:     cluster.ImageDownloader,
		// 		ImageNeuronRTD:      cluster.ImageNeuronRTD,
		// 		ImageRequestMonitor: cluster.ImageRequestMonitor,
		// 	},
		// 	APIVersion:          consts.CortexVersion,
		// 	IsOperatorInCluster: ,
		// 	OperatorID:          hashedAccountID,
		// 	ClusterID:           ,
		// }

		// TODO create cloudwatch dashboard here if it doesn't exist already
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
