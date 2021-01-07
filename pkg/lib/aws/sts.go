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
	"github.com/cortexlabs/cortex/pkg/lib/hash"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
)

// Returns account ID, whether the credentials were valid, any other error that occurred
// Ignores cache, so will re-run on every call to this method
func (c *Client) CheckCredentials() (string, string, error) {
	response, err := c.STS().GetCallerIdentity(nil)
	if err != nil {
		return "", "", ErrorInvalidAWSCredentials(err)
	}

	c.accountID = response.Account
	c.hashedAccountID = pointer.String(hash.String(*c.accountID))

	return *c.accountID, *c.hashedAccountID, nil
}

// Only re-checks the credentials if they have never been checked (so will not catch e.g. credentials expiring or getting revoked)
func (c *Client) GetCachedAccountID() (string, string, error) {
	if c.accountID == nil || c.hashedAccountID == nil {
		if _, _, err := c.CheckCredentials(); err != nil {
			return "", "", err
		}
	}
	return *c.accountID, *c.hashedAccountID, nil
}
