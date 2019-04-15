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
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	cc "github.com/cortexlabs/cortex/pkg/operator/cortexconfig"
)

func AuthUser(accessKeyID string, secretAccessKey string) (bool, error) {
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(cc.Region),
		DisableSSL:  aws.Bool(false),
		Credentials: credentials.NewStaticCredentials(accessKeyID, secretAccessKey, ""),
	})
	if err != nil {
		return false, errors.WithStack(err)
	}
	userSTSClient := sts.New(sess)

	response, err := userSTSClient.GetCallerIdentity(nil)
	if awsErr, ok := err.(awserr.RequestFailure); ok {
		if awsErr.StatusCode() == 403 {
			return false, nil
		}
	}
	if err != nil {
		return false, errors.WithStack(err)
	}

	return *response.Account == awsAccountID, nil
}
