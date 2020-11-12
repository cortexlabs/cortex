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

package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	container "cloud.google.com/go/container/apiv1"
	"github.com/cortexlabs/cortex/pkg/consts"
	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/debug"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
	"github.com/cortexlabs/yaml"
	"github.com/spf13/cobra"
	containerpb "google.golang.org/genproto/googleapis/container/v1"
)

var _gcpCmd = &cobra.Command{
	Use: "gcp",
	Run: func(cmd *cobra.Command, args []string) {

		if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") == "" {
			exit.Error(errors.ErrorUnexpected("need to specify $GOOGLE_APPLICATION_CREDENTIALS"))
		}

		ctx := context.Background()

		gcpPath := args[0]

		awsClusterConfigPath := args[1]

		gcpBytes, err := files.ReadFileBytes(gcpPath)
		if err != nil {
			exit.Error(err)
		}

		gcpConfig := clusterconfig.GCPConfig{}
		err = yaml.Unmarshal(gcpBytes, &gcpConfig)
		if err != nil {
			exit.Error(err)
		}

		accessConfig, err := clusterconfig.DefaultAccessConfig()
		if err != nil {
			exit.Error(err)
		}

		errs := cr.ParseYAMLFile(accessConfig, clusterconfig.AccessValidation, awsClusterConfigPath)
		if errors.HasError(errs) {
			exit.Error(errors.Append(errors.FirstError(errs...), fmt.Sprintf("\n\ncluster configuration schema can be found here: https://docs.cortex.dev/v/%s/cluster-management/config", consts.CortexVersionMinor)))
		}

		awsCreds := AWSCredentials{
			AWSAccessKeyID:            os.Getenv("AWS_ACCESS_KEY_ID"),
			AWSSecretAccessKey:        os.Getenv("AWS_SECRET_ACCESS_KEY"),
			ClusterAWSAccessKeyID:     os.Getenv("AWS_ACCESS_KEY_ID"),
			ClusterAWSSecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
		}

		clusterConfig := &clusterconfig.Config{}

		err = clusterconfig.SetDefaults(clusterConfig)
		if err != nil {
			exit.Error(err)
		}

		errs = cr.ParseYAMLFile(clusterConfig, clusterconfig.Validation, awsClusterConfigPath)
		if errors.HasError(errs) {
			exit.Error(errors.Append(errors.FirstError(errs...), fmt.Sprintf("\n\ncluster configuration schema can be found here: https://docs.cortex.dev/v/%s/cluster-management/config", consts.CortexVersionMinor)))
		}

		c, err := container.NewClusterManagerClient(ctx)
		if err != nil {
			exit.Error(err)
		}

		parent := fmt.Sprintf("projects/%s/locations/%s", gcpConfig.Project, gcpConfig.Zone)
		clusterName := gcpConfig.ClusterName
		request := containerpb.CreateClusterRequest{
			Parent: parent,
			Cluster: &containerpb.Cluster{
				Name: clusterName,
				// LoggingService: "none",
				// MonitoringService: "none",
				NodePools: []*containerpb.NodePool{
					{
						Name: "ng-cortex-operator",
						Config: &containerpb.NodeConfig{
							MachineType: "n1-standard-2",
						},
						InitialNodeCount: 1,
					},
					{
						Name: "ng-cortex-worker-on-demand",
						Config: &containerpb.NodeConfig{
							MachineType: gcpConfig.InstanceType,
							Labels: map[string]string{
								"workload": "true",
							},
							Taints: []*containerpb.NodeTaint{
								{
									Key:    "workload",
									Value:  "true",
									Effect: containerpb.NodeTaint_NO_SCHEDULE,
								},
							},
							// Accelerators: []*containerpb.AcceleratorConfig{
							// 	{
							// 		AcceleratorCount: 1,
							// 		AcceleratorType:  "nvidia-tesla-k80",
							// 	},
							// },
						},
						Autoscaling: &containerpb.NodePoolAutoscaling{
							Enabled:      true,
							MinNodeCount: int32(gcpConfig.MinInstances),
							MaxNodeCount: int32(gcpConfig.MaxInstances),
						},
						InitialNodeCount: 1,
					},
					// AddonsConfig: &containerpb.AddonsConfig{
					// 	HorizontalPodAutoscaling: &containerpb.HorizontalPodAutoscaling{
					// 		Disabled: true,
					// 	},
				},
				Locations: []string{"us-central1-a"},
			},
		}

		_, err = c.CreateCluster(ctx, &request)
		if err != nil {
			exit.Error(err)
		}

		for {
			time.Sleep(10 * time.Second)
			req := &containerpb.GetClusterRequest{
				Name: fmt.Sprintf("%s/clusters/%s", parent, clusterName),
			}

			resp, err := c.GetCluster(ctx, req)
			if err != nil {
				debug.Ppg(resp)
				debug.Pp(err.Error())
				continue
			} else {
				debug.Pp(resp.Status)
			}

			if resp.Status != containerpb.Cluster_PROVISIONING {
				break
			}

			fmt.Println("\n")
		}
		_, _, err = runManagerWithClusterConfigGCP("/root/install.sh", &gcpConfig, clusterConfig, awsCreds, nil, nil)
		if err != nil {
			exit.Error(err)
		}
	},
}
