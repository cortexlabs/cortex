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

package aws

import (
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
)

var EKSSupportedRegions strset.Set

func init() {
	EKSSupportedRegions = strset.New()
	for region := range InstanceMetadatas {
		EKSSupportedRegions.Add(region)
	}
}

// Returns info for the first cluster that matches all provided tags, or nil if there are no matching clusters
func (c *Client) GetEKSCluster(tags map[string]string) (*eks.Cluster, error) {
	var cluster *eks.Cluster
	var fnErr error
	err := c.EKS().ListClustersPages(&eks.ListClustersInput{},
		func(page *eks.ListClustersOutput, lastPage bool) bool {
			for _, clusterName := range page.Clusters {
				clusterInfo, err := c.EKS().DescribeCluster(&eks.DescribeClusterInput{Name: clusterName})
				if err != nil {
					fnErr = errors.WithStack(err)
					return false
				}

				missingTag := false
				for key, value := range tags {
					if v, ok := clusterInfo.Cluster.Tags[key]; !ok || v == nil || *v != value {
						missingTag = true
						break
					}
				}

				if !missingTag {
					cluster = clusterInfo.Cluster
					return false
				}
			}

			return true
		})

	if err != nil {
		return nil, errors.WithStack(err)
	}
	if fnErr != nil {
		return nil, fnErr
	}

	return cluster, nil
}
