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
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/hash"
)

// Returns account ID, whether the credentials were valid, any other error that occurred
func (c *Client) VerifyAccountID() (bool, error) {
	if c.stsClient == nil {
		c.stsClient = sts.New(c.sess)
	}

	response, err := c.stsClient.GetCallerIdentity(nil)
	if awsErr, ok := err.(awserr.RequestFailure); ok {
		if awsErr.StatusCode() == 403 {
			return false, nil
		}
	}
	if err != nil {
		return false, errors.WithStack(err)
	}
	c.AccountID = *response.Account
	c.HashedAccountID = hash.String(c.AccountID)
	return true, nil
}
