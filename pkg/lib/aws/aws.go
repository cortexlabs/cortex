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

package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sts"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/hash"
)

var Client *client

type client struct {
	Bucket   string
	LogGroup string
	Region   string

	s3Client             *s3.S3
	stsClient            *sts.STS
	cloudWatchLogsClient *cloudwatchlogs.CloudWatchLogs
	awsAccountID         string
	HashedAccountID      string
}

func Init(bucket, logGroup, region string) *client {
	sess := session.Must(session.NewSession(&aws.Config{
		Region:     aws.String(region),
		DisableSSL: aws.Bool(false),
	}))

	c := &client{
		Bucket:               bucket,
		LogGroup:             logGroup,
		Region:               region,
		s3Client:             s3.New(sess),
		stsClient:            sts.New(sess),
		cloudWatchLogsClient: cloudwatchlogs.New(sess),
	}
	response, err := c.stsClient.GetCallerIdentity(nil)
	if err != nil {
		errors.Exit(err, ErrorAuth())
	}
	c.awsAccountID = *response.Account
	c.HashedAccountID = hash.String(c.awsAccountID)

	return c
}
