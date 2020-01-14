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
	"fmt"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
)

type ErrorKind int

const (
	ErrUnknown ErrorKind = iota
	ErrInvalidS3aPath
	ErrInvalidS3Path
	ErrAuth
	ErrBucketInaccessible
	ErrInstanceTypeLimitIsZero
	ErrNoValidSpotPrices
	ErrReadCredentials
)

var _errorKinds = []string{
	"err_unknown",
	"err_invalid_s3a_path",
	"err_invalid_s3_path",
	"err_auth",
	"err_bucket_inaccessible",
	"err_instance_type_limit_is_zero",
	"err_no_valid_spot_prices",
	"err_read_credentials",
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

type Error struct {
	Kind    ErrorKind
	message string
}

func (e Error) Error() string {
	return e.message
}

func ErrorInvalidS3aPath(provided string) error {
	return errors.WithStack(Error{
		Kind:    ErrInvalidS3aPath,
		message: fmt.Sprintf("%s is not a valid s3a path (e.g. s3a://cortex-examples/iris-classifier/tensorflow is a valid s3a path)", s.UserStr(provided)),
	})
}

func ErrorInvalidS3Path(provided string) error {
	return errors.WithStack(Error{
		Kind:    ErrInvalidS3Path,
		message: fmt.Sprintf("%s is not a valid s3 path (e.g. s3://cortex-examples/iris-classifier/tensorflow is a valid s3 path)", s.UserStr(provided)),
	})
}

func ErrorAuth() error {
	return errors.WithStack(Error{
		Kind:    ErrAuth,
		message: "unable to authenticate with AWS",
	})
}

func ErrorBucketInaccessible(bucket string) error {
	return errors.WithStack(Error{
		Kind:    ErrBucketInaccessible,
		message: fmt.Sprintf("bucket \"%s\" not found or insufficient permissions", bucket),
	})
}

func ErrorInstanceTypeLimitIsZero(instanceType string, region string) error {
	return errors.WithStack(Error{
		Kind:    ErrInstanceTypeLimitIsZero,
		message: fmt.Sprintf(`you don't have access to %s instances in %s; please request access in the appropriate region (https://console.aws.amazon.com/support/cases#/create?issueType=service-limit-increase&limitType=ec2-instances)"`, instanceType, region),
	})
}

func ErrorNoValidSpotPrices(instanceType string, region string) error {
	return errors.WithStack(Error{
		Kind:    ErrNoValidSpotPrices,
		message: fmt.Sprintf("no spot prices were found for %s instances in %s", instanceType, region),
	})
}

func ErrorReadCredentials() error {
	return errors.WithStack(Error{
		Kind:    ErrReadCredentials,
		message: "unable to read AWS credentials from credentials file",
	})
}
