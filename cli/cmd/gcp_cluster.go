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
	"cloud.google.com/go/storage"
	"github.com/cortexlabs/cortex/pkg/consts"
	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	"github.com/cortexlabs/cortex/pkg/lib/files"
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
	ProjectId    string `json:"project_id"`
}

// TODO: remove references to AWS here
func upGCP(gcpPath string, awsClusterConfigPath string) {
	if awsClusterConfigPath == "" {
		exit.Error(errors.ErrorUnexpected("usage: cortex cluster up -c gcp.yaml --backup-aws aws-cluster.yaml"))
	}

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

	if os.Getenv("AWS_ACCESS_KEY_ID") == "" {
		exit.Error(errors.ErrorUnexpected("need to specify $AWS_ACCESS_KEY_ID"))
	}

	if os.Getenv("AWS_SECRET_ACCESS_KEY") == "" {
		exit.Error(errors.ErrorUnexpected("need to specify $AWS_SECRET_ACCESS_KEY"))
	}

	awsCreds := AWSCredentials{
		AWSAccessKeyID:            os.Getenv("AWS_ACCESS_KEY_ID"),
		AWSSecretAccessKey:        os.Getenv("AWS_SECRET_ACCESS_KEY"),
		ClusterAWSAccessKeyID:     os.Getenv("AWS_ACCESS_KEY_ID"),
		ClusterAWSSecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
	}

	clusterConfig := &clusterconfig.Config{}

	errs := cr.ParseYAMLFile(clusterConfig, clusterconfig.Validation, awsClusterConfigPath)
	if errors.HasError(errs) {
		exit.Error(errors.Append(errors.FirstError(errs...), fmt.Sprintf("\n\ncluster configuration schema can be found here: https://docs.cortex.dev/v/%s/cluster-management/config", consts.CortexVersionMinor)))
	}

	c, err := container.NewClusterManagerClient(ctx)
	if err != nil {
		exit.Error(err)
	}

	bucketID := hash.String(gcpCredentials.ClientEmail + gcpConfig.Zone)[:10]

	defaultBucket := gcpConfig.ClusterName + "-" + bucketID

	err = createGCPBucketIfNotFound(defaultBucket, gcpConfig.Project)
	if err != nil {
		exit.Error(err)
	}

	gcpConfig.Bucket = defaultBucket

	fmt.Print("spinning up a cluster .")

	parent := fmt.Sprintf("projects/%s/locations/%s", gcpConfig.Project, gcpConfig.Zone)
	clusterName := gcpConfig.ClusterName
	request := containerpb.CreateClusterRequest{
		Parent: parent,
		Cluster: &containerpb.Cluster{
			Name: clusterName,
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
			Name: fmt.Sprintf("%s/clusters/%s", parent, clusterName),
		}

		resp, err := c.GetCluster(ctx, req)
		if err != nil {
			exit.Error(err)
		}

		if resp.Status != containerpb.Cluster_PROVISIONING {
			fmt.Println(" ✓")
			break
		}

		fmt.Print(".")
	}

	_, _, err = runManagerWithClusterConfigGCP("/root/install.sh", &gcpConfig, clusterConfig, awsCreds, nil, nil)
	if err != nil {
		exit.Error(err)
	}
	exit.Ok()
}

func createGCPBucketIfNotFound(bucketName string, projectID string) error {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}

	bucket := client.Bucket(bucketName)
	_, err = bucket.Attrs(ctx)
	if err == storage.ErrBucketNotExist {
		fmt.Print("￮ creating a new gcs bucket: ", bucketName)
		err = bucket.Create(ctx, projectID, nil)
		if err != nil {
			return err
		}
		fmt.Println("  ✓")
		return nil
	}
	fmt.Println("￮ using existing gcs bucket: ", bucketName+"  ✓")
	return nil
}
