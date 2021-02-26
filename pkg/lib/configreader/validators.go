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

package configreader

import (
	"regexp"
	"strings"
	"time"

	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/docker"
	"github.com/cortexlabs/cortex/pkg/lib/files"
)

var _emailRegex *regexp.Regexp

func init() {
	_emailRegex = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
}

func GetFilePathValidator(baseDir string) func(string) (string, error) {
	return func(val string) (string, error) {
		if err := files.CheckFile(val); err != nil {
			return "", err
		}

		return val, nil
	}
}

func S3aPathValidator(val string) (string, error) {
	if !aws.IsValidS3aPath(val) {
		return "", aws.ErrorInvalidS3aPath(val)
	}
	return val, nil
}

func S3PathValidator(val string) (string, error) {
	if !aws.IsValidS3Path(val) {
		return "", aws.ErrorInvalidS3Path(val)
	}
	return val, nil
}

func EmailValidator(val string) (string, error) {
	if len(val) > 320 {
		return "", ErrorEmailTooLong()
	}

	if !_emailRegex.MatchString(val) {
		return "", ErrorEmailInvalid()
	}

	return val, nil
}

type DurationValidation struct {
	GreaterThan          *time.Duration
	GreaterThanOrEqualTo *time.Duration
	LessThan             *time.Duration
	LessThanOrEqualTo    *time.Duration
	MultipleOf           *time.Duration
}

func DurationParser(v *DurationValidation) func(string) (interface{}, error) {
	return func(str string) (interface{}, error) {
		d, err := time.ParseDuration(str)
		if err != nil {
			return nil, err
		}

		if v == nil {
			return d, nil
		}

		if v.GreaterThan != nil {
			if d <= *v.GreaterThan {
				return nil, ErrorMustBeGreaterThan(str, *v.GreaterThan)
			}
		}

		if v.GreaterThanOrEqualTo != nil {
			if d < *v.GreaterThanOrEqualTo {
				return nil, ErrorMustBeGreaterThanOrEqualTo(str, *v.GreaterThanOrEqualTo)
			}
		}

		if v.LessThan != nil {
			if d >= *v.LessThan {
				return nil, ErrorMustBeLessThan(str, *v.LessThan)
			}
		}

		if v.LessThanOrEqualTo != nil {
			if d > *v.LessThanOrEqualTo {
				return nil, ErrorMustBeLessThanOrEqualTo(str, *v.LessThanOrEqualTo)
			}
		}

		if v.MultipleOf != nil {
			if d.Nanoseconds()%(*v.MultipleOf).Nanoseconds() != 0 {
				return nil, ErrorIsNotMultiple(d, *v.MultipleOf)
			}
		}

		return d, nil
	}
}

func ValidateImageVersion(image, cortexVersion string) (string, error) {
	if !strings.HasPrefix(image, "quay.io/cortexlabs/") && !strings.HasPrefix(image, "quay.io/cortexlabsdev/") && !strings.HasPrefix(image, "cortexlabs/") && !strings.HasPrefix(image, "cortexlabsdev/") {
		return image, nil
	}

	tag := docker.ExtractImageTag(image)
	// in docker, missing tag implies "latest"
	if tag == "" {
		tag = "latest"
	}

	if !strings.HasPrefix(tag, cortexVersion) {
		return "", ErrorImageVersionMismatch(image, tag, cortexVersion)
	}

	return image, nil
}
