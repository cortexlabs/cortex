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
	"github.com/aws/aws-sdk-go/service/acm"
	"github.com/aws/aws-sdk-go/service/apigatewayv2"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/servicequotas"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sts"
)

type clients struct {
	s3             *s3.S3
	s3Uploader     *s3manager.Uploader
	s3Downloader   *s3manager.Downloader
	sts            *sts.STS
	sqs            *sqs.SQS
	ec2            *ec2.EC2
	elbv2          *elbv2.ELBV2
	eks            *eks.EKS
	ecr            *ecr.ECR
	acm            *acm.ACM
	autoscaling    *autoscaling.AutoScaling
	cloudWatchLogs *cloudwatchlogs.CloudWatchLogs
	cloudWatch     *cloudwatch.CloudWatch
	apiGatewayV2   *apigatewayv2.ApiGatewayV2
	serviceQuotas  *servicequotas.ServiceQuotas
	cloudFormation *cloudformation.CloudFormation
	iam            *iam.IAM
}

func (c *Client) S3() *s3.S3 {
	if c.clients.s3 == nil {
		c.clients.s3 = s3.New(c.sess)
	}
	return c.clients.s3
}

func (c *Client) S3Uploader() *s3manager.Uploader {
	if c.clients.s3Uploader == nil {
		c.clients.s3Uploader = s3manager.NewUploader(c.sess)
	}
	return c.clients.s3Uploader
}

func (c *Client) S3Downloader() *s3manager.Downloader {
	if c.clients.s3Downloader == nil {
		c.clients.s3Downloader = s3manager.NewDownloader(c.sess)
	}
	return c.clients.s3Downloader
}

func (c *Client) STS() *sts.STS {
	if c.clients.sts == nil {
		c.clients.sts = sts.New(c.sess)
	}
	return c.clients.sts
}

func (c *Client) SQS() *sqs.SQS {
	if c.clients.sqs == nil {
		c.clients.sqs = sqs.New(c.sess)
	}
	return c.clients.sqs
}

func (c *Client) EC2() *ec2.EC2 {
	if c.clients.ec2 == nil {
		c.clients.ec2 = ec2.New(c.sess)
	}
	return c.clients.ec2
}

func (c *Client) ELBV2() *elbv2.ELBV2 {
	if c.clients.elbv2 == nil {
		c.clients.elbv2 = elbv2.New(c.sess)
	}
	return c.clients.elbv2
}

func (c *Client) EKS() *eks.EKS {
	if c.clients.eks == nil {
		c.clients.eks = eks.New(c.sess)
	}
	return c.clients.eks
}

func (c *Client) ECR() *ecr.ECR {
	if c.clients.ecr == nil {
		c.clients.ecr = ecr.New(c.sess)
	}
	return c.clients.ecr
}

func (c *Client) CloudFormation() *cloudformation.CloudFormation {
	if c.clients.cloudFormation == nil {
		c.clients.cloudFormation = cloudformation.New(c.sess)
	}
	return c.clients.cloudFormation
}

func (c *Client) Autoscaling() *autoscaling.AutoScaling {
	if c.clients.autoscaling == nil {
		c.clients.autoscaling = autoscaling.New(c.sess)
	}
	return c.clients.autoscaling
}

func (c *Client) ACM() *acm.ACM {
	if c.clients.acm == nil {
		c.clients.acm = acm.New(c.sess)
	}
	return c.clients.acm
}

func (c *Client) CloudWatchLogs() *cloudwatchlogs.CloudWatchLogs {
	if c.clients.cloudWatchLogs == nil {
		c.clients.cloudWatchLogs = cloudwatchlogs.New(c.sess)
	}
	return c.clients.cloudWatchLogs
}

func (c *Client) CloudWatch() *cloudwatch.CloudWatch {
	if c.clients.cloudWatch == nil {
		c.clients.cloudWatch = cloudwatch.New(c.sess)
	}
	return c.clients.cloudWatch
}

func (c *Client) APIGatewayV2() *apigatewayv2.ApiGatewayV2 {
	if c.clients.apiGatewayV2 == nil {
		c.clients.apiGatewayV2 = apigatewayv2.New(c.sess)
	}
	return c.clients.apiGatewayV2
}

func (c *Client) ServiceQuotas() *servicequotas.ServiceQuotas {
	if c.clients.serviceQuotas == nil {
		c.clients.serviceQuotas = servicequotas.New(c.sess)
	}
	return c.clients.serviceQuotas
}

func (c *Client) IAM() *iam.IAM {
	if c.clients.iam == nil {
		c.clients.iam = iam.New(c.sess)
	}
	return c.clients.iam
}
