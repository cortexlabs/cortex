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

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
)

const (
	ErrInvalidAWSCredentials        = "aws.invalid_aws_credentials"
	ErrInvalidS3aPath               = "aws.invalid_s3a_path"
	ErrInvalidS3Path                = "aws.invalid_s3_path"
	ErrUnexpectedMissingCredentials = "aws.unexpected_missing_credentials"
	ErrAuth                         = "aws.auth"
	ErrBucketInaccessible           = "aws.bucket_inaccessible"
	ErrBucketNotFound               = "aws.bucket_not_found"
	ErrInsufficientInstanceQuota    = "aws.insufficient_instance_quota"
	ErrNoValidSpotPrices            = "aws.no_valid_spot_prices"
	ErrReadCredentials              = "aws.read_credentials"
	ErrECRExtractingCredentials     = "aws.ecr_failed_credentials"
	ErrDashboardWidthOutOfRange     = "aws.dashboard_width_ouf_of_range"
	ErrDashboardHeightOutOfRange    = "aws.dashboard_height_out_of_range"
	ErrRegionNotConfigured          = "aws.region_not_configured"
	ErrUnableToFindCredentials      = "aws.unable_to_find_credentials"
	ErrNATGatewayLimitExceeded      = "aws.nat_gateway_limit_exceeded"
	ErrEIPLimitExceeded             = "aws.eip_limit_exceeded"
	ErrInternetGatewayLimitExceeded = "aws.internet_gateway_limit_exceeded"
	ErrVPCLimitExceeded             = "aws.vpc_limit_exceeded"
)

func IsAWSError(err error) bool {
	if _, ok := errors.CauseOrSelf(err).(awserr.Error); ok {
		return true
	}
	return false
}

func IsNotFoundErr(err error) bool {
	return IsErrCode(err, "NotFound")
}

func IsNoSuchKeyErr(err error) bool {
	return IsErrCode(err, s3.ErrCodeNoSuchKey)
}

func IsNoSuchEntityErr(err error) bool {
	return IsErrCode(err, iam.ErrCodeNoSuchEntityException)
}

func IsNoSuchBucketErr(err error) bool {
	return IsErrCode(err, s3.ErrCodeNoSuchBucket)
}

func IsNonExistentQueueErr(err error) bool {
	return IsErrCode(err, sqs.ErrCodeQueueDoesNotExist)
}

func IsGenericNotFoundErr(err error) bool {
	return IsNotFoundErr(err) || IsNoSuchKeyErr(err) || IsNoSuchBucketErr(err)
}

func IsErrCode(err error, errorCode string) bool {
	awsErr, ok := errors.CauseOrSelf(err).(awserr.Error)
	if !ok {
		return false
	}
	if awsErr.Code() == errorCode {
		return true
	}
	return false
}

func ErrorInvalidAWSCredentials(awsErr error) error {
	awsErrMsg := errors.Message(awsErr)
	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidAWSCredentials,
		Message: "invalid AWS credentials\n" + awsErrMsg,
		Cause:   awsErr,
	})
}

func ErrorInvalidS3aPath(provided string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidS3aPath,
		Message: fmt.Sprintf("%s is not a valid s3a path (e.g. s3a://cortex-examples/pytorch/iris-classifier/weights.pth is a valid s3a path)", s.UserStr(provided)),
	})
}

func ErrorInvalidS3Path(provided string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidS3Path,
		Message: fmt.Sprintf("%s is not a valid s3 path (e.g. s3://cortex-examples/pytorch/iris-classifier/weights.pth is a valid s3 path)", s.UserStr(provided)),
	})
}

func ErrorUnexpectedMissingCredentials(awsAccessKeyID string, awsSecretAccessKey string) error {
	var msg string
	if awsAccessKeyID == "" && awsSecretAccessKey == "" {
		msg = "aws access key id and aws secret access key are missing"
	} else if awsAccessKeyID == "" {
		msg = "aws access key id is missing"
	} else if awsSecretAccessKey == "" {
		msg = "aws secret access key is missing"
	}

	return errors.WithStack(&errors.Error{
		Kind:    ErrUnexpectedMissingCredentials,
		Message: msg,
	})
}

func ErrorAuth() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrAuth,
		Message: "unable to authenticate with AWS",
	})
}

func ErrorBucketInaccessible(bucket string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrBucketInaccessible,
		Message: fmt.Sprintf("bucket \"%s\" is not accessible with the specified AWS credentials", bucket),
	})
}

func ErrorBucketNotFound(bucket string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrBucketNotFound,
		Message: fmt.Sprintf("bucket \"%s\" not found", bucket),
	})
}

func ErrorInsufficientInstanceQuota(instanceTypes []string, lifecycle string, region string, requiredVCPUs int64, vCPUQuota int64, quotaCode string) error {
	url := fmt.Sprintf("https://%s.console.aws.amazon.com/servicequotas/home?region=%s#!/services/ec2/quotas/%s", region, region, quotaCode)
	andInstanceTypes := s.StrsAnd(instanceTypes)
	return errors.WithStack(&errors.Error{
		Kind:    ErrInsufficientInstanceQuota,
		Message: fmt.Sprintf("your cluster may require up to %d vCPU %s %s instances, but your AWS quota for %s %s instances in %s is only %d vCPU; please reduce the maximum number of %s %s instances your cluster may use (e.g. by changing max_instances and/or spot_config if applicable), or request a quota increase to at least %d vCPU here: %s (if your request was recently approved, please allow ~30 minutes for AWS to reflect this change)", requiredVCPUs, lifecycle, andInstanceTypes, lifecycle, andInstanceTypes, region, vCPUQuota, lifecycle, andInstanceTypes, requiredVCPUs, url),
	})
}

func ErrorNoValidSpotPrices(instanceType string, region string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrNoValidSpotPrices,
		Message: fmt.Sprintf("no spot prices were found for %s instances in %s", instanceType, region),
	})
}

func ErrorReadCredentials() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrReadCredentials,
		Message: "unable to read AWS credentials from credentials file",
	})
}

func ErrorECRExtractingCredentials() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrECRExtractingCredentials,
		Message: "unable to extract ECR credentials",
	})
}

func ErrorDashboardWidthOutOfRange(width int) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrDashboardWidthOutOfRange,
		Message: fmt.Sprintf("dashboard width %d out of range; width must be between %d and %d", width, _dashboardMinWidthUnits, _dashboardMaxWidthUnits),
	})
}

func ErrorDashboardHeightOutOfRange(height int) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrDashboardHeightOutOfRange,
		Message: fmt.Sprintf("dashboard height %d out of range; height must be between %d and %d", height, _dashboardMinHeightUnits, _dashboardMaxHeightUnits),
	})
}

func ErrorRegionNotConfigured() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrRegionNotConfigured,
		Message: "aws region has not been configured; please set a default region (e.g. `export AWS_REGION=us-west-2`)",
	})
}

func ErrorUnableToFindCredentials() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrUnableToFindCredentials,
		Message: "unable to find aws credentials; instructions about configuring aws credentials can be found at https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-quickstart.html",
	})
}

func ErrorNATGatewayLimitExceeded(currentLimit, additionalQuotaRequired int, availabilityZones []string, region string) error {
	url := "https://console.aws.amazon.com/servicequotas/home?#!/services/vpc/quotas"
	return errors.WithStack(&errors.Error{
		Kind:    ErrNATGatewayLimitExceeded,
		Message: fmt.Sprintf("NAT gateway limit of %d exceeded in availability zones %s of region %s; remove some of the existing NAT gateways or increase your quota for NAT gateways by at least %d here: %s (if your request was recently approved, please allow ~30 minutes for AWS to reflect this change)", currentLimit, s.StrsAnd(availabilityZones), region, additionalQuotaRequired, url),
	})
}

func ErrorEIPLimitExceeded(currentLimit, additionalQuotaRequired int, region string) error {
	url := "https://console.aws.amazon.com/servicequotas/home?#!/services/ec2/quotas"
	return errors.WithStack(&errors.Error{
		Kind:    ErrEIPLimitExceeded,
		Message: fmt.Sprintf("elastic IPs limit of %d exceeded in region %s; remove some of the existing elastic IPs or increase your quota for elastic IPs by at least %d here: %s (if your request was recently approved, please allow ~30 minutes for AWS to reflect this change)", currentLimit, region, additionalQuotaRequired, url),
	})
}

func ErrorInternetGatewayLimitExceeded(currentLimit, additionalQuotaRequired int, region string) error {
	url := "https://console.aws.amazon.com/servicequotas/home?#!/services/vpc/quotas"
	return errors.WithStack(&errors.Error{
		Kind:    ErrInternetGatewayLimitExceeded,
		Message: fmt.Sprintf("internet gateway limit of %d exceeded in region %s; remove some of the existing internet gateways or increase your quota for internet gateways by at least %d here: %s (if your request was recently approved, please allow ~30 minutes for AWS to reflect this change)", currentLimit, region, additionalQuotaRequired, url),
	})
}

func ErrorVPCLimitExceeded(currentLimit, additionalQuotaRequired int, region string) error {
	url := "https://console.aws.amazon.com/servicequotas/home?#!/services/vpc/quotas"
	return errors.WithStack(&errors.Error{
		Kind:    ErrVPCLimitExceeded,
		Message: fmt.Sprintf("VPC limit of %d exceeded in region %s; remove some of the existing VPCs or increase your quota for VPCs by at least %d here: %s (if your request was recently approved, please allow ~30 minutes for AWS to reflect this change)", currentLimit, region, additionalQuotaRequired, url),
	})
}
