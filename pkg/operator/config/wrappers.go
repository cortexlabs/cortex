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
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/gcp"
	"github.com/cortexlabs/cortex/pkg/types"
)

func ClusterName() string {
	switch Provider {
	case types.AWSProviderType:
		return Cluster.ClusterName
	case types.GCPProviderType:
		return GCPCluster.ClusterName
	}
	return ""
}

func Bucket() string {
	switch Provider {
	case types.AWSProviderType:
		return Cluster.Bucket
	case types.GCPProviderType:
		return GCPCluster.Bucket
	}
	return ""
}

func BucketPath(key string) string {
	switch Provider {
	case types.AWSProviderType:
		return aws.S3Path(Cluster.Bucket, key)
	case types.GCPProviderType:
		return gcp.GCSPath(GCPCluster.Bucket, key)
	}
	return ""
}

func ReadJSONFromBucket(objPtr interface{}, key string) error {
	switch Provider {
	case types.AWSProviderType:
		return AWS.ReadJSONFromS3(objPtr, Cluster.Bucket, key)
	case types.GCPProviderType:
		return GCP.ReadJSONFromGCS(objPtr, GCPCluster.Bucket, key)
	}
	return nil
}

func IsBucketFile(fileKey string) (bool, error) {
	switch Provider {
	case types.AWSProviderType:
		return AWS.IsS3File(Cluster.Bucket, fileKey)
	case types.GCPProviderType:
		return GCP.IsGCSFile(GCPCluster.Bucket, fileKey)
	}
	return false, nil
}

func UploadBytesToBucket(data []byte, key string) error {
	switch Provider {
	case types.AWSProviderType:
		return AWS.UploadBytesToS3(data, Cluster.Bucket, key)
	case types.GCPProviderType:
		return GCP.UploadBytesToGCS(data, GCPCluster.Bucket, key)
	}
	return nil
}

func UploadJSONToBucket(obj interface{}, key string) error {
	switch Provider {
	case types.AWSProviderType:
		return AWS.UploadJSONToS3(obj, Cluster.Bucket, key)
	case types.GCPProviderType:
		return GCP.UploadJSONToGCS(obj, GCPCluster.Bucket, key)
	}
	return nil
}

func ListBucketDirOneLevel(dir string, maxResults *int64) ([]string, error) {
	switch Provider {
	case types.AWSProviderType:
		return AWS.ListS3DirOneLevel(Cluster.Bucket, dir, maxResults)
	case types.GCPProviderType:
		return GCP.ListGCSDirOneLevel(GCPCluster.Bucket, dir, maxResults)
	}
	return nil, nil
}

func DeleteBucketDir(dir string, continueIfFailure bool) error {
	switch Provider {
	case types.AWSProviderType:
		return AWS.DeleteS3Dir(Cluster.Bucket, dir, continueIfFailure)
	case types.GCPProviderType:
		return GCP.DeleteGCSDir(GCPCluster.Bucket, dir, continueIfFailure)
	}
	return nil
}
