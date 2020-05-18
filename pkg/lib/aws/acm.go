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
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/acm"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
)

func (c *Client) VerifyCertificate(sslCertificateARN string) (bool, error) {
	_, err := c.ACM().DescribeCertificate(&acm.DescribeCertificateInput{
		CertificateArn: aws.String(sslCertificateARN),
	})

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == "ResourceNotFoundException" {
				return false, ErrorSSLCertificateARNNotFound(sslCertificateARN, c.Region)
			}
		}
		return false, errors.Wrap(err, sslCertificateARN)
	}

	return true, nil
}
