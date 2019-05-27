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
	"context"
	"net/url"

	sparkop "github.com/GoogleCloudPlatform/spark-on-k8s-operator/pkg/apis/sparkoperator.k8s.io/v1alpha1"
	"gocloud.dev/blob"
	_ "gocloud.dev/blob/fileblob"
	_ "gocloud.dev/blob/s3blob"
	corev1 "k8s.io/api/core/v1"

	"github.com/cortexlabs/cortex/pkg/lib/cloud/aws"
	"github.com/cortexlabs/cortex/pkg/lib/cloud/local"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
)

type Client struct {
	ProviderType ProviderType
	bucket       *blob.Bucket
	Provider
}

type Provider interface {
	GetLogs(string) (string, error)
	AuthUser(string) (bool, error)
	HashedAccountID() string
	OperatorBucketURL() *url.URL
	WorkloadBucketURL() *url.URL

	IsValidHadoopPath(string) (bool, error)
	SparkEnvCredentials() map[string]sparkop.NameKey
	EnvCredentials() []corev1.EnvVar
	StorageVolumes() []corev1.Volume
	StorageVolumeMounts() []corev1.VolumeMount
	APIsBaseURL(kubernetes *k8s.Client) (string, error)
}

type Options struct {
	Bucket             string  `json:"bucket"`
	Region             *string `json:"region"`
	LogGroup           *string `json:"log_group"`
	OperatorLocalMount *string `json:"operator_local_mount"`
	OperatorInCluster  bool    `json:"operator_in_cluster"`
}

func newAWSCloud(opts *Options) (*Client, error) {
	if opts.Region == nil {
		return nil, ErrorRequiredFieldNotDefined("region")
	}
	if opts.LogGroup == nil {
		return nil, ErrorRequiredFieldNotDefined("log_group")
	}

	clientLayer := aws.New(opts.Bucket, *opts.Region, *opts.LogGroup)
	ctx := context.Background()

	bucket, err := blob.OpenBucket(ctx, clientLayer.OperatorBucketURL().String())
	if err != nil {
		return nil, err
	}
	return &Client{
		ProviderType: AWSProviderType,
		bucket:       bucket,
		Provider:     clientLayer,
	}, nil
}

func newLocalCloud(opts *Options) (*Client, error) {
	clientLayer := local.New(opts.Bucket, opts.OperatorInCluster, opts.OperatorLocalMount)
	ctx := context.Background()
	bucket, err := blob.OpenBucket(ctx, clientLayer.OperatorBucketURL().String())
	if err != nil {
		return nil, err
	}

	return &Client{
		ProviderType: LocalProviderType,
		bucket:       bucket,
		Provider:     clientLayer,
	}, nil
}

func New(cloudProvider string, opts *Options) (*Client, error) {
	cloudProviderKey := ProviderTypeFromString(cloudProvider)
	if cloudProviderKey == AWSProviderType {
		return newAWSCloud(opts)
	} else if cloudProviderKey == LocalProviderType {
		return newLocalCloud(opts)
	} else {
		return nil, ErrorUnsupportedProviderType(cloudProvider)
	}
}

func (c *Client) GetProviderType() string {
	return c.ProviderType.String()
}

func (c *Client) Close() {
	if c.bucket != nil {
		c.bucket.Close()
	}
}
