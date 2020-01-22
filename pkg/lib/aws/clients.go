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
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/servicequotas"
	"github.com/aws/aws-sdk-go/service/sts"
)

type clients struct {
	s3                *s3.S3
	sts               *sts.STS
	ec2               *ec2.EC2
	autoscaling       *autoscaling.AutoScaling
	cloudWatchLogs    *cloudwatchlogs.CloudWatchLogs
	cloudWatchMetrics *cloudwatch.CloudWatch
	serviceQuotas     *servicequotas.ServiceQuotas
}

func (c *Client) S3() *s3.S3 {
	if c.clients.s3 == nil {
		c.clients.s3 = s3.New(c.sess)
	}
	return c.clients.s3
}

func (c *Client) STS() *sts.STS {
	if c.clients.sts == nil {
		c.clients.sts = sts.New(c.sess)
	}
	return c.clients.sts
}

func (c *Client) EC2() *ec2.EC2 {
	if c.clients.ec2 == nil {
		c.clients.ec2 = ec2.New(c.sess)
	}
	return c.clients.ec2
}

func (c *Client) Autoscaling() *autoscaling.AutoScaling {
	if c.clients.autoscaling == nil {
		c.clients.autoscaling = autoscaling.New(c.sess)
	}
	return c.clients.autoscaling
}

func (c *Client) CloudWatchLogs() *cloudwatchlogs.CloudWatchLogs {
	if c.clients.cloudWatchLogs == nil {
		c.clients.cloudWatchLogs = cloudwatchlogs.New(c.sess)
	}
	return c.clients.cloudWatchLogs
}

func (c *Client) CloudWatchMetrics() *cloudwatch.CloudWatch {
	if c.clients.cloudWatchMetrics == nil {
		c.clients.cloudWatchMetrics = cloudwatch.New(c.sess)
	}
	return c.clients.cloudWatchMetrics
}

func (c *Client) ServiceQuotas() *servicequotas.ServiceQuotas {
	if c.clients.serviceQuotas == nil {
		c.clients.serviceQuotas = servicequotas.New(c.sess)
	}
	return c.clients.serviceQuotas
}
