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

package clusterconfig

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"

	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
)

func DefaultPolicyName(clusterName string, region string) string {
	return fmt.Sprintf("cortex-%s-%s", clusterName, region)
}

func DefaultPolicyARN(accountID string, clusterName string, region string) string {
	return fmt.Sprintf("arn:aws:iam::%s:policy/%s", accountID, DefaultPolicyName(clusterName, region))
}

var _cortexPolicy = `
{
	"Version": "2012-10-17",
	"Statement": [
		{
			"Action": [
				"sts:GetCallerIdentity",
				"ecr:GetAuthorizationToken",
				"ecr:BatchGetImage",
				"sqs:ListQueues"
			],
			"Effect": "Allow",
			"Resource": "*"
		},
		{
			"Effect": "Allow",
			"Action": "sqs:*",
			"Resource": "arn:aws:sqs:{{ .Region }}:{{ .AccountID }}:cx-*"
		},
		{
			"Effect": "Allow",
			"Action": "s3:*",
			"Resource": "arn:aws:s3:::{{ .Bucket }}"
		},
		{
			"Effect": "Allow",
			"Action": "s3:*",
			"Resource": "arn:aws:s3:::{{ .Bucket }}/*"
		},
		{
			"Effect": "Allow",
			"Action": [
				"logs:CreateLogStream",
				"logs:DescribeLogStreams",
				"logs:PutLogEvents",
				"logs:CreateLogGroup"
			],
			"Resource": "arn:aws:logs:{{ .Region }}:{{ .AccountID }}:log-group:{{ .LogGroup }}:*"
		},
		{
			"Effect": "Allow",
			"Action": "logs:CreateLogGroup",
			"Resource": "arn:aws:logs:{{ .Region }}:{{ .AccountID }}:log-group:{{ .LogGroup }}"
		}
	]
}
`

type CortexPolicyTemplateArgs struct {
	ClusterName string
	LogGroup    string
	Region      string
	Bucket      string
	AccountID   string
}

func CreateDefaultPolicy(awsClient *aws.Client, args CortexPolicyTemplateArgs) error {
	policyName := DefaultPolicyName(args.ClusterName, args.Region)
	accountID, _, err := awsClient.GetCachedAccountID()
	if err != nil {
		return err
	}

	policyARN := DefaultPolicyARN(accountID, args.ClusterName, args.Region)
	policyTemplate, err := template.New("policy").Parse(_cortexPolicy)
	if err != nil {
		return errors.Wrap(err, "failed to parse aws policy template")
	}

	buf := &bytes.Buffer{}
	err = policyTemplate.Execute(buf, args)
	if err != nil {
		return errors.Wrap(err, "failed to execute aws policy template")
	}

	compactBuf := &bytes.Buffer{}

	err = json.Compact(compactBuf, buf.Bytes())
	if err != nil {
		return errors.Wrap(err, "failed to parse and remove whitespace from aws policy json")
	}

	policyDocument := compactBuf.String()

	_, err = awsClient.IAM().CreatePolicy(&iam.CreatePolicyInput{
		PolicyDocument: &policyDocument,
		PolicyName:     &policyName,
	})
	if err != nil {
		if aws.IsErrCode(err, iam.ErrCodeEntityAlreadyExistsException) {
			err := AddNewPolicyVersion(awsClient, policyARN, policyDocument)
			if err != nil {
				return errors.Wrap(err, "failed to create iam policy for cortex")
			}
		} else {
			return errors.Wrap(err, "failed to create iam policy for cortex")
		}
	}

	return nil
}

func AddNewPolicyVersion(awsClient *aws.Client, policyARN string, policyDocument string) error {
	policies, err := awsClient.IAM().ListPolicyVersions(&iam.ListPolicyVersionsInput{
		PolicyArn: &policyARN,
	})
	if err != nil {
		return err
	}

	if len(policies.Versions) == 0 {
		return errors.ErrorUnexpected("encountered a policy without any policy versions")
	}

	numPolicies := len(policies.Versions)
	oldestPolicy := *policies.Versions[0]

	for _, policyPtr := range policies.Versions {
		policy := *policyPtr

		if policy.CreateDate.Before(*oldestPolicy.CreateDate) && !*policy.IsDefaultVersion {
			oldestPolicy = policy
		}
	}

	// can only have a max of 5 versions, so delete the oldest non-default version before adding a new policy verion
	if numPolicies > 4 {
		_, err := awsClient.IAM().DeletePolicyVersion(&iam.DeletePolicyVersionInput{
			PolicyArn: &policyARN,
			VersionId: oldestPolicy.VersionId,
		})
		if err != nil {
			return errors.WithStack(err)
		}
	}

	_, err = awsClient.IAM().CreatePolicyVersion(&iam.CreatePolicyVersionInput{
		SetAsDefault:   pointer.Bool(true),
		PolicyDocument: &policyDocument,
		PolicyArn:      &policyARN,
	})
	if err != nil {
		return errors.WithStack(err)
	}

	return nil

}
