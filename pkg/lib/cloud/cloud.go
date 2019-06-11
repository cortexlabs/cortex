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

package cloud

import (
	"bytes"

	sparkop "github.com/GoogleCloudPlatform/spark-on-k8s-operator/pkg/apis/sparkoperator.k8s.io/v1alpha1"
	_ "gocloud.dev/blob/fileblob"
	_ "gocloud.dev/blob/s3blob"
	corev1 "k8s.io/api/core/v1"

	"github.com/cortexlabs/cortex/pkg/lib/k8s"
)

type Client interface {
	GetLogs(string) (string, error)
	AuthUser(string) (bool, error)
	HashedAccountID() string
	Close()
	// OperatorBucketURL() *url.URL
	// WorkloadBucketURL() *url.URL

	ExternalPrefixExists(string) (bool, error)
	IsValidHadoopPath(string) (bool, error)
	SparkEnvCredentials() map[string]sparkop.NameKey
	EnvCredentials() []corev1.EnvVar
	StorageVolumes() []corev1.Volume
	StorageVolumeMounts() []corev1.VolumeMount
	APIsBaseURL(kubernetes *k8s.Client) (string, error)

	BucketPath(key string) string
	FileExists(key string) (bool, error)
	PutBytes(data []byte, key string) error
	PutFile(filePath string, key string) error
	PutBuffer(buffer *bytes.Buffer, key string) error
	PutString(str string, key string) error
	PutJSON(obj interface{}, key string) error
	PutMsgpack(obj interface{}, key string) error
	GetJSON(objPtr interface{}, key string) error
	GetBytes(key string) ([]byte, error)
	GetMsgpack(objPtr interface{}, key string) error
	GetString(key string) (string, error)
	DeleteByPrefix(prefix string, continueIfFailure bool) error
}

type Config struct {
	Bucket             string  `json:"bucket"`
	Region             *string `json:"region"`
	LogGroup           *string `json:"log_group"`
	OperatorLocalMount *string `json:"operator_local_mount"`
	OperatorInCluster  bool    `json:"operator_in_cluster"`
	OperatorHostIP     *string `json:"operator_host_ip"`
}

// func newAWSCloud(opts *Options) (*Client, error) {
// 	if opts.Region == nil {
// 		return nil, ErrorRequiredFieldNotDefined("region")
// 	}
// 	if opts.LogGroup == nil {
// 		return nil, ErrorRequiredFieldNotDefined("log_group")
// 	}

// 	provider := aws.New(opts.Bucket, *opts.Region, *opts.LogGroup)
// 	ctx := context.Background()

// 	//bucket, err := blob.OpenBucket(ctx, provider.OperatorBucketURL().String())
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &Client{
// 		ProviderType: AWSProviderType,
// 		bucket:       bucket,
// 		Provider:     provider,
// 	}, nil
// }

// func newLocalCloud(opts *Options) (*Client, error) {
// 	debug.Ppj(opts)
// 	provider := local.New(opts.Bucket, opts.OperatorInCluster, opts.OperatorLocalMount, opts.OperatorHostIP)
// 	ctx := context.Background()
// 	//bucket, err := blob.OpenBucket(ctx, provider.OperatorBucketURL().String())
// 	if err != nil {
// 		return nil, err
// 	}

// 	return &Client{
// 		ProviderType: LocalProviderType,
// 		bucket:       bucket,
// 		Provider:     provider,
// 	}, nil
// }

// func New(providerType ProviderType, opts *Options) (*Client, error) {
// 	switch providerType {
// 	case AWSProviderType:
// 		return newAWSCloud(opts)
// 	case LocalProviderType:
// 		return newLocalCloud(opts)
// 	default:
// 		return nil, ErrorUnsupportedProviderType(providerType.String())
// 	}

// }
