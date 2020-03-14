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

type ErrorKind int

const (
	ErrUnknown ErrorKind = iota
	ErrInvalidAWSCredentials
	ErrInvalidS3aPath
	ErrInvalidS3Path
	ErrAuth
	ErrBucketInaccessible
	ErrBucketNotFound
	ErrInstanceTypeLimitIsZero
	ErrNoValidSpotPrices
	ErrReadCredentials
)

var _errorKinds = []string{
	"aws.unknown",
	"aws.invalid_aws_credentials",
	"aws.invalid_s3a_path",
	"aws.invalid_s3_path",
	"aws.auth",
	"aws.bucket_inaccessible",
	"aws.bucket_not_found",
	"aws.instance_type_limit_is_zero",
	"aws.no_valid_spot_prices",
	"aws.read_credentials",
}

var _ = [1]int{}[int(ErrReadCredentials)-(len(_errorKinds)-1)] // Ensure list length matches

func (t ErrorKind) String() string {
	return _errorKinds[t]
}

// MarshalText satisfies TextMarshaler
func (t ErrorKind) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// UnmarshalText satisfies TextUnmarshaler
func (t *ErrorKind) UnmarshalText(text []byte) error {
	enum := string(text)
	for i := 0; i < len(_errorKinds); i++ {
		if enum == _errorKinds[i] {
			*t = ErrorKind(i)
			return nil
		}
	}

	*t = ErrUnknown
	return nil
}

// UnmarshalBinary satisfies BinaryUnmarshaler
// Needed for msgpack
func (t *ErrorKind) UnmarshalBinary(data []byte) error {
	return t.UnmarshalText(data)
}

// MarshalBinary satisfies BinaryMarshaler
func (t ErrorKind) MarshalBinary() ([]byte, error) {
	return []byte(t.String()), nil
}

func IsNotFoundErr(err error) bool {
	return CheckErrCode(err, "NotFound")
}

func IsNoSuchKeyErr(err error) bool {
	return CheckErrCode(err, "NoSuchKey")
}

func IsNoSuchBucketErr(err error) bool {
	return CheckErrCode(err, "NoSuchBucket")
}

func IsGenericNotFoundErr(err error) bool {
	return IsNotFoundErr(err) || IsNoSuchKeyErr(err) || IsNoSuchBucketErr(err)
}

func CheckErrCode(err error, errorCode string) bool {
	awsErr, ok := errors.Cause(err).(awserr.Error)
	if !ok {
		return false
	}
	if awsErr.Code() == errorCode {
		return true
	}
	return false
}

func ErrorInvalidAWSCredentials() error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrInvalidAWSCredentials,
		Message: "invalid AWS credentials",
	})
}

func ErrorInvalidS3aPath(provided string) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrInvalidS3aPath,
		Message: fmt.Sprintf("%s is not a valid s3a path (e.g. s3a://cortex-examples/iris-classifier/tensorflow is a valid s3a path)", s.UserStr(provided)),
	})
}

func ErrorInvalidS3Path(provided string) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrInvalidS3Path,
		Message: fmt.Sprintf("%s is not a valid s3 path (e.g. s3://cortex-examples/iris-classifier/tensorflow is a valid s3 path)", s.UserStr(provided)),
	})
}

func ErrorAuth() error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrAuth,
		Message: "unable to authenticate with AWS",
	})
}

func ErrorBucketInaccessible(bucket string) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrBucketInaccessible,
		Message: fmt.Sprintf("bucket \"%s\" not found or insufficient permissions", bucket),
	})
}

func ErrorBucketNotFound(bucket string) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrBucketNotFound,
		Message: fmt.Sprintf("bucket \"%s\" not found", bucket),
	})
}

func ErrorInstanceTypeLimitIsZero(instanceType string, region string) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrInstanceTypeLimitIsZero,
		Message: fmt.Sprintf(`you don't have access to %s instances in %s; please request access in the appropriate region (https://console.aws.amazon.com/support/cases#/create?issueType=service-limit-increase&limitType=ec2-instances). If you submitted a request and it was recently approved, please allow ~30 minutes for AWS to reflect this change."`, instanceType, region),
		User:    true,
	})
}

func ErrorNoValidSpotPrices(instanceType string, region string) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrNoValidSpotPrices,
		Message: fmt.Sprintf("no spot prices were found for %s instances in %s", instanceType, region),
		User:    true,
	})
}

func ErrorReadCredentials() error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrReadCredentials,
		Message: "unable to read AWS credentials from credentials file",
	})
}
