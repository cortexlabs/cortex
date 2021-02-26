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
	"fmt"

	"github.com/aws/aws-sdk-go/aws/credentials"
)

// access key ID may be unavailable depending on how the client was instantiated
func (c *Client) AccessKeyID() *string {
	if c.sess.Config.Credentials == nil {
		return nil
	}

	sessCreds, err := c.sess.Config.Credentials.Get()
	if err != nil {
		return nil
	}

	if sessCreds.AccessKeyID == "" {
		return nil
	}

	return &sessCreds.AccessKeyID
}

func (c *Client) SecretAccessKey() *string {
	if c.sess.Config.Credentials == nil {
		return nil
	}

	sessCreds, err := c.sess.Config.Credentials.Get()
	if err != nil {
		return nil
	}

	if sessCreds.SecretAccessKey == "" {
		return nil
	}

	return &sessCreds.SecretAccessKey
}

func (c *Client) SessionToken() *string {
	if c.sess.Config.Credentials == nil {
		return nil
	}

	sessCreds, err := c.sess.Config.Credentials.Get()
	if err != nil {
		return nil
	}

	if sessCreds.SessionToken == "" {
		return nil
	}

	return &sessCreds.SessionToken
}

func GetCredentialsFromCLIConfigFile() (string, string, error) {
	creds := credentials.NewSharedCredentials("", "")
	if creds == nil {
		return "", "", ErrorReadCredentials()
	}

	value, err := creds.Get()
	if err != nil {
		return "", "", err
	}

	if value.SessionToken != "" {
		fmt.Println("warning: credentials requiring aws session tokens are not supported")
	}

	if value.AccessKeyID == "" || value.SecretAccessKey == "" {
		return "", "", ErrorReadCredentials()
	}

	return value.AccessKeyID, value.SecretAccessKey, nil
}
