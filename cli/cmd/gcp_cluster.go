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
	"encoding/json"
	"fmt"
	"os"
	"time"

	container "cloud.google.com/go/container/apiv1"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/gcp"
	"github.com/cortexlabs/cortex/pkg/lib/hash"
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
	containerpb "google.golang.org/genproto/googleapis/container/v1"
	"gopkg.in/yaml.v2"
)

type GCPCredentials struct {
	ClientEmail  string `json:"client_email"`
	ClientId     string `json:"client_id"`
	PrivateKeyId string `json:"private_key_id"`
	PrivateKey   string `json:"private_key"`
	ProjectId    string `json:"project_id"` // TODO validate that this project id matches GCP cluster config's Project field
}

func upGCP(gcpPath string) {
	if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") == "" {
		exit.Error(errors.ErrorUnexpected("need to specify $GOOGLE_APPLICATION_CREDENTIALS"))
	}

	ctx := context.Background()

	gcpBytes, err := files.ReadFileBytes(gcpPath)
	if err != nil {
		exit.Error(err)
	}

	credsBytes, err := files.ReadFileBytes(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"))
	if err != nil {
		exit.Error(err)
	}

	gcpCredentials := GCPCredentials{}
	err = json.Unmarshal(credsBytes, &gcpCredentials)
	if err != nil {
		exit.Error(err)
	}

	gcpConfig := clusterconfig.GCPConfig{}
	err = yaml.Unmarshal(gcpBytes, &gcpConfig)
	if err != nil {
		exit.Error(err)
	}

	c, err := container.NewClusterManagerClient(ctx)
	if err != nil {
		exit.Error(err)
	}

	bucketID := hash.String(gcpConfig.Project + gcpConfig.Zone)[:10]

	if gcpConfig.Bucket == "" {
		gcpConfig.Bucket = gcpConfig.ClusterName + "-" + bucketID
	}

	GCP := &gcp.Client{
		ProjectID: gcpConfig.Project,
		Zone:      gcpConfig.Zone,
	}

	err = gcpConfig.Validate(GCP)
	if err != nil {
		exit.Error(err)
	}

	nodeLabels := map[string]string{"workload": "true"}
	var accelerators []*containerpb.AcceleratorConfig

	if gcpConfig.AcceleratorType != nil {
		accelerators = append(accelerators, &containerpb.AcceleratorConfig{
			AcceleratorCount: 1,
			AcceleratorType:  *gcpConfig.AcceleratorType,
		})
		nodeLabels["nvidia.com/gpu"] = "present"
	}

	err = GCP.CreateBucket(gcpConfig.Bucket, gcpConfig.Project, true)
	if err != nil {
		exit.Error(err)
	}

	fmt.Print("￮ spinning up a cluster .")

	parent := fmt.Sprintf("projects/%s/locations/%s", gcpConfig.Project, gcpConfig.Zone)
	request := containerpb.CreateClusterRequest{
		Parent: parent,
		Cluster: &containerpb.Cluster{
			Name: gcpConfig.ClusterName,
			NodePools: []*containerpb.NodePool{
				{
					Name: "ng-cortex-operator",
					Config: &containerpb.NodeConfig{
						MachineType: "n1-standard-2",
						OauthScopes: []string{
							"https://www.googleapis.com/auth/compute",
							"https://www.googleapis.com/auth/devstorage.read_only",
						},
						ServiceAccount: gcpCredentials.ClientEmail,
					},
					InitialNodeCount: 1,
				},
				{
					Name: "ng-cortex-worker-on-demand",
					Config: &containerpb.NodeConfig{
						MachineType: gcpConfig.InstanceType,
						Labels:      nodeLabels,
						Taints: []*containerpb.NodeTaint{
							{
								Key:    "workload",
								Value:  "true",
								Effect: containerpb.NodeTaint_NO_SCHEDULE,
							},
						},
						Accelerators: accelerators,
						OauthScopes: []string{
							"https://www.googleapis.com/auth/compute",
							"https://www.googleapis.com/auth/devstorage.read_only",
						},
						ServiceAccount: gcpCredentials.ClientEmail,
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
			Locations: []string{gcpConfig.Zone},
		},
	}

	_, err = c.CreateCluster(ctx, &request)
	if err != nil {
		exit.Error(err)
	}

	for {
		time.Sleep(10 * time.Second)
		req := &containerpb.GetClusterRequest{
			Name: fmt.Sprintf("%s/clusters/%s", parent, gcpConfig.ClusterName),
		}

		resp, err := c.GetCluster(ctx, req)
		if err != nil {
			exit.Error(err)
		}

		if resp.Status == containerpb.Cluster_ERROR {
			fmt.Println(" ✗")
			helpStr := fmt.Sprintf("\nyour cluster couldn't be spun up; here is the error that was encountered: %s", resp.StatusMessage)
			helpStr += fmt.Sprintf("\nadditional error information may be found on the cluster's page in the GCP console: https://console.cloud.google.com/kubernetes/clusters/details/%s/%s?project=%s", gcpConfig.Zone, gcpConfig.ClusterName, gcpConfig.Project)
			fmt.Println(helpStr)
			exit.Error(ErrorClusterUp(resp.StatusMessage))
		}

		if resp.Status != containerpb.Cluster_PROVISIONING {
			fmt.Println(" ✓")
			break
		}

		fmt.Print(".")
	}

	_, _, err = runManagerWithClusterConfigGCP("/root/install.sh", &gcpConfig, nil, nil)
	if err != nil {
		exit.Error(err)
	}
	exit.Ok()
}
