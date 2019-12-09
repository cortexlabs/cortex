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
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/servicequotas"
)

func VerifyInstanceQuota(accessKeyID, secretAccessKey, region, instanceType string) error {
	if !strings.HasPrefix(instanceType, "p") {
		return nil
	}

	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(region),
		DisableSSL:  aws.Bool(false),
		Credentials: credentials.NewStaticCredentials(accessKeyID, secretAccessKey, ""),
	})
	if err != nil {
		return err
	}
	svc := servicequotas.New(sess)

	pFamilyCPULimit := 0
	err = svc.ListServiceQuotasPages(
		&servicequotas.ListServiceQuotasInput{
			ServiceCode: aws.String("ec2"),
		},
		func(page *servicequotas.ListServiceQuotasOutput, lastPage bool) bool {
			if page == nil {
				return false
			}
			for _, quota := range page.Quotas {
				if quota == nil {
					continue
				}
				if *quota.QuotaName == "Running On-Demand P instances" {
					pFamilyCPULimit = int(*quota.Value) // quota is specified in number of vCPU permitted per family
					return false
				}
			}
			return true
		},
	)
	if err != nil {
		return err
	}

	if pFamilyCPULimit == 0 {
		return ErrorPFamilyInstanceUseNotPermitted(region)
	}

	return nil
}
