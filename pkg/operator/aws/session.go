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

	s "github.com/cortexlabs/cortex/pkg/api/strings"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/hash"
	cc "github.com/cortexlabs/cortex/pkg/operator/cortexconfig"
)

var awsAccountID string
var s3Client *s3.S3
var stsClient *sts.STS
var cloudWatchLogsClient *cloudwatchlogs.CloudWatchLogs
var HashedAccountID string

func init() {
	sess := session.Must(session.NewSession(&aws.Config{
		Region:     aws.String(cc.Region),
		DisableSSL: aws.Bool(false),
	}))

	s3Client = s3.New(sess)
	cloudWatchLogsClient = cloudwatchlogs.New(sess)
	stsClient = sts.New(sess)

	response, err := stsClient.GetCallerIdentity(nil)
	if err != nil {
		errors.Exit(err, s.ErrUnableToAuthAws)
	}
	awsAccountID = *response.Account
	HashedAccountID = hash.String(awsAccountID)
}
