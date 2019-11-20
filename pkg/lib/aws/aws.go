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

//go:generate python3 gen_instance_metadata.py
//go:generate gofmt -s -w instance_metadata.go

package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sts"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/hash"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
)

type Client struct {
	Region               string
	Bucket               string
	S3                   *s3.S3
	stsClient            *sts.STS
	autoscaling          *autoscaling.AutoScaling
	CloudWatchLogsClient *cloudwatchlogs.CloudWatchLogs
	CloudWatchMetrics    *cloudwatch.CloudWatch
	AccountID            string
	HashedAccountID      string
}

var EKSSupportedRegions strset.Set

func init() {
	EKSSupportedRegions = strset.New()
	for region := range InstanceMetadatas {
		EKSSupportedRegions.Add(region)
	}
}

func New(region string, bucket string, withAccountID bool) (*Client, error) {
	sess := session.Must(session.NewSession(&aws.Config{
		Region:     aws.String(region),
		DisableSSL: aws.Bool(false),
	}))

	awsClient := &Client{
		Bucket:               bucket,
		Region:               region,
		S3:                   s3.New(sess),
		stsClient:            sts.New(sess),
		autoscaling:          autoscaling.New(sess),
		CloudWatchMetrics:    cloudwatch.New(sess),
		CloudWatchLogsClient: cloudwatchlogs.New(sess),
	}

	if withAccountID {
		response, err := awsClient.stsClient.GetCallerIdentity(nil)
		if err != nil {
			return nil, errors.Wrap(err, ErrorAuth().Error())
		}
		awsClient.AccountID = *response.Account
		awsClient.HashedAccountID = hash.String(awsClient.AccountID)
	}

	return awsClient, nil
}

func NewFromS3Path(s3Path string, withAccountID bool) (*Client, error) {
	bucket, _, err := SplitS3Path(s3Path)
	if err != nil {
		return nil, err
	}

	region, err := GetBucketRegion(bucket)
	if err != nil {
		return nil, err
	}

	return New(region, bucket, withAccountID)
}
