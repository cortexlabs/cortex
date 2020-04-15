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

package local

import (
	"fmt"

	"github.com/cortexlabs/cortex/cli/types/cliconfig"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
)

func Refresh(env cliconfig.Environment, apiName string) (schema.RefreshResponse, error) {
	var awsClient *aws.Client
	var err error
	if env.AWSAccessKeyID != nil {
		awsClient, err = aws.NewFromCreds(*env.AWSRegion, *env.AWSAccessKeyID, *env.AWSSecretAccessKey)
		if err != nil {
			return schema.RefreshResponse{}, err
		}
	} else {
		awsClient, err = aws.NewAnonymousClient()
		if err != nil {
			return schema.RefreshResponse{}, err
		}
	}
	CacheModel(apiName, awsClient)
	// apiSpec, err := FindAPISpec(apiName)
	// if err != nil {
	// 	return schema.RefreshResponse{}, nil
	// }

	// var awsClient *aws.Client
	// if env.AWSAccessKeyID != nil {
	// 	awsClient, err = aws.NewFromCreds(*env.AWSRegion, *env.AWSAccessKeyID, *env.AWSSecretAccessKey)
	// 	if err != nil {
	// 		return schema.RefreshResponse{}, err
	// 	}
	// } else {
	// 	awsClient, err = aws.NewAnonymousClient()
	// 	if err != nil {
	// 		return schema.RefreshResponse{}, err
	// 	}
	// }

	// err := DeleteContainers(apiName)
	// if err != nil {

	// }

	// UpdateAPI()

	return schema.RefreshResponse{
		Message: fmt.Sprintf("updating %s", apiName),
	}, nil
}
