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
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
)

func GetCredentialsFromCLIConfigFile() (string, string, error) {
	creds := credentials.NewSharedCredentials("", "")
	if creds == nil {
		return "", "", errors.New("unable to read AWS credentials from credentials file")
	}
	value, err := creds.Get()
	if err != nil {
		return "", "", err
	}

	if value.AccessKeyID == "" || value.SecretAccessKey == "" {
		return "", "", errors.New("unable to read AWS credentials from credentials file")
	}

	return value.AccessKeyID, value.SecretAccessKey, nil
}
