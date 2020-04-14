/*
Copyright 2020 Cortex Labs, Inc.

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
	"encoding/base64"
	"strings"

	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/regex"
)

type ECRAuthConfig struct {
	Username      string
	AccessToken   string
	ProxyEndpoint string
}

func (c *Client) GetECRAuthToken() (*ecr.GetAuthorizationTokenOutput, error) {
	result, err := c.ECR().GetAuthorizationToken(&ecr.GetAuthorizationTokenInput{})
	if err != nil {
		return result, errors.Wrap(err, "failed to retrieve ECR auth token")
	}
	return result, nil
}

func (c *Client) GetECRAuthConfig() (ECRAuthConfig, error) {
	tokenOutput, err := c.GetECRAuthToken()
	if err != nil {
		return ECRAuthConfig{}, err
	}
	authData := tokenOutput.AuthorizationData[0]

	credentials, err := base64.URLEncoding.DecodeString(*authData.AuthorizationToken)
	if err != nil {
		return ECRAuthConfig{}, errors.Wrap(err, ErrorECRExtractingCredentials().Error())
	}
	credentialsString := string(credentials)
	splitCredentials := strings.Split(credentialsString, ":")
	if len(splitCredentials) != 2 {
		return ECRAuthConfig{}, ErrorECRExtractingCredentials()
	}

	return ECRAuthConfig{
		Username:      splitCredentials[0],
		AccessToken:   splitCredentials[1],
		ProxyEndpoint: *authData.ProxyEndpoint,
	}, nil
}

func GetAccountIDFromECRURL(path string) string {
	if regex.IsValidECRURL(path) {
		return strings.Split(path, ".")[0]
	}
	return ""
}
