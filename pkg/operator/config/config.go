/*
Copyright 2019 Cortex Labs, Inc.

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
	"os"
	"strings"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/clusterconfig"
	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/hash"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
)

var (
	Cluster         *clusterconfig.InternalClusterConfig
	AWS             *aws.Client
	Kubernetes      *k8s.Client
	IstioKubernetes *k8s.Client
	Telemetry       *telemetry.Client
)

func Init() error {
	var err error

	Cluster = &clusterconfig.InternalClusterConfig{
		APIVersion:        consts.CortexVersion,
		OperatorInCluster: strings.ToLower(os.Getenv("CORTEX_OPERATOR_IN_CLUSTER")) != "false",
	}

	clusterConfigPath := os.Getenv("CORTEX_CLUSTER_CONFIG_PATH")
	if clusterConfigPath == "" {
		clusterConfigPath = consts.ClusterConfigPath
	}

	errs := cr.ParseYAMLFile(Cluster, clusterconfig.UserValidation, clusterConfigPath)
	if errors.HasErrors(errs) {
		return errors.FirstError(errs...)
	}

	Cluster.InstanceMetadata = aws.InstanceMetadatas[*Cluster.Region][*Cluster.InstanceType]

	if Kubernetes, err = k8s.New(consts.K8sNamespace, Cluster.OperatorInCluster); err != nil {
		return err
	}

	if IstioKubernetes, err = k8s.New("istio-system", Cluster.OperatorInCluster); err != nil {
		return err
	}

	Cluster.ID = hash.String(*Cluster.Bucket + *Cluster.Region + Cluster.LogGroup)

	AWS, err = aws.New(*Cluster.Region, *Cluster.Bucket, true)
	if err != nil {
		errors.Exit(err)
	}
	Telemetry = telemetry.New(consts.TelemetryURL, AWS.HashedAccountID, Cluster.Telemetry)

	return nil
}
