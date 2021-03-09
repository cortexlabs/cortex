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

package cmd

import (
	"fmt"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/aws"

	"github.com/cortexlabs/cortex/pkg/lib/prompt"
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
)

func newAWSClient(region string) (*aws.Client, error) {
	if err := clusterconfig.ValidateRegion(region); err != nil {
		return nil, err
	}

	awsClient, err := aws.NewForRegion(region)
	if err != nil {
		return nil, err
	}

	if _, _, err := awsClient.CheckCredentials(); err != nil {
		return nil, err
	}

	fmt.Println("using aws credentials with access key " + *awsClient.AccessKeyID() + "\n")

	return awsClient, nil
}

func promptIfNotAdmin(awsClient *aws.Client, disallowPrompt bool) {
	accessKeyMsg := ""
	if accessKey := awsClient.AccessKeyID(); accessKey != nil {
		accessKeyMsg = fmt.Sprintf(" (with access key %s)", *accessKey)
	}

	if !awsClient.IsAdmin() {
		warningStr := fmt.Sprintf("warning: your IAM user%s does not have administrator access. This will likely prevent Cortex from installing correctly, so it is recommended to attach the AdministratorAccess policy to your IAM user (or to a group that your IAM user belongs to) via the AWS IAM console. If you'd like, you may provide separate credentials for your cluster to use after it's running (see https://docs.cortex.dev/v/%s/).\n\n", accessKeyMsg, consts.CortexVersionMinor)
		if disallowPrompt {
			fmt.Print(warningStr)
		} else {
			prompt.YesOrExit(warningStr+"are you sure you want to continue without administrator access?", "", "")
		}
	}
}

func warnIfNotAdmin(awsClient *aws.Client) {
	accessKeyMsg := ""
	if accessKey := awsClient.AccessKeyID(); accessKey != nil {
		accessKeyMsg = fmt.Sprintf(" (with access key %s)", *accessKey)
	}

	if !awsClient.IsAdmin() {
		fmt.Println(fmt.Sprintf("warning: your IAM user or assumed role%s does not have administrator access. This may prevent this command from executing correctly, so it is recommended to attach the AdministratorAccess policy to your IAM user or role.", accessKeyMsg), "", "")
	}
}
