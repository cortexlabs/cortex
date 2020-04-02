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
	"fmt"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
)

const (
	ErrInvalidAWSCredentials   = "aws.invalid_aws_credentials"
	ErrInvalidS3aPath          = "aws.invalid_s3a_path"
	ErrInvalidS3Path           = "aws.invalid_s3_path"
	ErrAuth                    = "aws.auth"
	ErrBucketInaccessible      = "aws.bucket_inaccessible"
	ErrBucketNotFound          = "aws.bucket_not_found"
	ErrInstanceTypeLimitIsZero = "aws.instance_type_limit_is_zero"
	ErrNoValidSpotPrices       = "aws.no_valid_spot_prices"
	ErrReadCredentials         = "aws.read_credentials"
)

func IsNotFoundErr(err error) bool {
	return CheckErrCode(err, "NotFound")
}

func IsNoSuchKeyErr(err error) bool {
	return CheckErrCode(err, "NoSuchKey")
}

func IsNoSuchBucketErr(err error) bool {
	return CheckErrCode(err, "NoSuchBucket")
}

func IsForbiddenErr(err error) bool {
	return CheckErrCode(err, "Forbidden")
}

func IsGenericNotFoundErr(err error) bool {
	return IsNotFoundErr(err) || IsNoSuchKeyErr(err) || IsNoSuchBucketErr(err)
}

func CheckErrCode(err error, errorCode string) bool {
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
	awsErrMsg := errors.MessageFirstLine(awsErr)
	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidAWSCredentials,
		Message: "invalid AWS credentials\n" + awsErrMsg,
		Cause:   awsErr,
	})
}

func ErrorInvalidS3aPath(provided string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidS3aPath,
		Message: fmt.Sprintf("%s is not a valid s3a path (e.g. s3a://cortex-examples/iris-classifier/tensorflow is a valid s3a path)", s.UserStr(provided)),
	})
}

func ErrorInvalidS3Path(provided string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidS3Path,
		Message: fmt.Sprintf("%s is not a valid s3 path (e.g. s3://cortex-examples/iris-classifier/tensorflow is a valid s3 path)", s.UserStr(provided)),
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

func ErrorInstanceTypeLimitIsZero(instanceType string, region string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrInstanceTypeLimitIsZero,
		Message: fmt.Sprintf(`you don't have access to %s instances in %s; please request access in the appropriate region (https://console.aws.amazon.com/support/cases#/create?issueType=service-limit-increase&limitType=ec2-instances). If you submitted a request and it was recently approved, please allow ~30 minutes for AWS to reflect this change."`, instanceType, region),
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
