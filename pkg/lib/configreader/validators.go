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
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/urls"
)

var portRe *regexp.Regexp

func init() {
	portRe = regexp.MustCompile(`:[0-9]+$`)
}

type PathValidation struct {
	Required bool
	Default  string
	BaseDir  string
}

func GetFilePathValidation(v *PathValidation) *StringValidation {
	validator := func(val string) (string, error) {
		val = files.RelPath(val, v.BaseDir)
		if err := files.CheckFile(val); err != nil {
			return "", err
		}

		return val, nil
	}

	return &StringValidation{
		Required:  v.Required,
		Default:   v.Default,
		Validator: validator,
	}
}

type S3aPathValidation struct {
	Required bool
	Default  string
}

func GetS3aPathValidation(v *S3aPathValidation) *StringValidation {
	validator := func(val string) (string, error) {
		if !aws.IsValidS3aPath(val) {
			return "", aws.ErrorInvalidS3aPath(val)
		}
		return val, nil
	}

	return &StringValidation{
		Required:  v.Required,
		Default:   v.Default,
		Validator: validator,
	}
}

type URLValidation struct {
	Required    bool
	Default     string
	DefaultHTTP bool // Otherwise default is https
	AddPort     bool
}

func GetURLValidation(v *URLValidation) *StringValidation {
	validator := func(val string) (string, error) {
		urlStr := strings.TrimSpace(val)

		if !strings.HasPrefix(strings.ToLower(urlStr), "http") {
			if v.DefaultHTTP {
				urlStr = "http://" + urlStr
			} else {
				urlStr = "https://" + urlStr
			}
		}

		if v.AddPort {
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

	return &StringValidation{
		Required:  v.Required,
		Default:   v.Default,
		Validator: validator,
	}
}
