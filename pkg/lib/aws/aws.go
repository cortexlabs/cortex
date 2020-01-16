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

//go:generate python3 gen_resource_metadata.py
//go:generate gofmt -s -w resource_metadata.go

package aws

import (
	"sort"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"

	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/servicequotas"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
)

type Credentials struct {
	AWSAccessKeyID           string `json:"aws_access_key_id"`
	AWSSecretAccessKey       string `json:"aws_secret_access_key"`
	CortexAWSAccessKeyID     string `json:"cortex_aws_access_key_id"`
	CortexAWSSecretAccessKey string `json:"cortex_aws_secret_access_key"`
}

type Client struct {
	Region               string
	Bucket               string
	Credentials          *Credentials
	sess                 *session.Session
	S3                   *s3.S3
	stsClient            *sts.STS
	ec2                  *ec2.EC2
	autoscaling          *autoscaling.AutoScaling
	CloudWatchLogsClient *cloudwatchlogs.CloudWatchLogs
	CloudWatchMetrics    *cloudwatch.CloudWatch
	servicequotas        *servicequotas.ServiceQuotas
	AccountID            string
	HashedAccountID      string
}

var EKSSupportedRegions strset.Set
var EKSSupportedRegionsSlice []string

func init() {
	EKSSupportedRegions = strset.New()
	for region := range InstanceMetadatas {
		EKSSupportedRegions.Add(region)
	}
	EKSSupportedRegionsSlice = EKSSupportedRegions.Slice()
	sort.Strings(EKSSupportedRegionsSlice)
}

func (c *Client) InitializeBucket(bucket string) error {

	bucketLocation, err := GetBucketRegion(bucket)
	if err != nil {
		return err
	}

	bucketSess := session.Must(session.NewSession(&aws.Config{
		Region:     aws.String(bucketLocation),
		DisableSSL: aws.Bool(false),
	}))

	c.Bucket = bucket
	c.S3 = s3.New(bucketSess)
	return nil
}

func newClient(sess *session.Session) *Client {
	return &Client{
		sess:                 sess,
		Region:               *sess.Config.Region,
		stsClient:            sts.New(sess),
		ec2:                  ec2.New(sess),
		servicequotas:        servicequotas.New(sess),
		autoscaling:          autoscaling.New(sess),
		CloudWatchMetrics:    cloudwatch.New(sess),
		CloudWatchLogsClient: cloudwatchlogs.New(sess),
	}
}

func NewInferCreds(region string) *Client {
	sess := session.Must(session.NewSession(&aws.Config{
		Region:     aws.String(region),
		DisableSSL: aws.Bool(false),
	}))

	return newClient(sess)
}

func NewFromCreds(creds Credentials, region string) *Client {
	sess := session.Must(session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(creds.AWSAccessKeyID, creds.AWSSecretAccessKey, ""),
		Region:      aws.String(region),
		DisableSSL:  aws.Bool(false),
	}))

	client := newClient(sess)
	client.Credentials = &creds
	return client
}

func NewFromS3Path(s3Path string) (*Client, error) {
	bucket, _, err := SplitS3Path(s3Path)
	if err != nil {
		return nil, err
	}

	region, err := GetBucketRegion(bucket)
	if err != nil {
		return nil, err
	}

	awsClient := NewInferCreds(region)

	if err = awsClient.InitializeBucket(bucket); err != nil {
		return nil, err
	}

	return awsClient, nil
}
