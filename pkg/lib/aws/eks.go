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

// Returns info for the cluster, or nil of no cluster exists with the provided name
func (c *Client) EKSClusterOrNil(clusterName string) (*eks.Cluster, error) {
	clusterInfo, err := c.EKS().DescribeCluster(&eks.DescribeClusterInput{Name: &clusterName})
	if err != nil {
		if IsErrCode(err, eks.ErrCodeResourceNotFoundException) {
			return nil, nil
		}
		return nil, errors.WithStack(err)
	}

	return clusterInfo.Cluster, nil
}
