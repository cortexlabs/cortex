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

package configreader

import (
	"regexp"
	"strings"

	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/urls"
)

var portRe *regexp.Regexp
var emailRegex *regexp.Regexp

func init() {
	portRe = regexp.MustCompile(`:[0-9]+$`)
	emailRegex = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
}

func GetFilePathValidator(baseDir string) func(string) (string, error) {
	return func(val string) (string, error) {
		val = files.RelPath(val, baseDir)
		if err := files.CheckFile(val); err != nil {
			return "", err
		}

		return val, nil
	}
}

func GetS3aPathValidator() func(string) (string, error) {
	return func(val string) (string, error) {
		if !aws.IsValidS3aPath(val) {
			return "", aws.ErrorInvalidS3aPath(val)
		}
		return val, nil
	}
}

func S3PathValidator() func(string) (string, error) {
	return func(val string) (string, error) {
		if !aws.IsValidS3Path(val) {
			return "", aws.ErrorInvalidS3Path(val)
		}
		return val, nil
	}
}

func EmailValidator() func(string) (string, error) {
	return func(val string) (string, error) {
		if len(val) > 320 {
			return "", errors.New("email exceeds max-length")
		}

		if !emailRegex.MatchString(val) {
			return "", errors.New("invalid email address")
		}

		return val, nil
	}
}

// uses https unless defaultHTTP == true
func GetURLValidator(defaultHTTP bool, addPort bool) func(string) (string, error) {
	return func(val string) (string, error) {
		urlStr := strings.TrimSpace(val)

		if !strings.HasPrefix(strings.ToLower(urlStr), "http") {
			if defaultHTTP {
				urlStr = "http://" + urlStr
			} else {
				urlStr = "https://" + urlStr
			}
		}

		if addPort {
			if !portRe.MatchString(urlStr) {
				if strings.HasPrefix(strings.ToLower(urlStr), "https") {
					urlStr = urlStr + ":443"
				} else {
					urlStr = urlStr + ":80"
				}
			}
		}

		if _, err := urls.Parse(urlStr); err != nil {
			return "", err
		}

		return urlStr, nil
	}
}
