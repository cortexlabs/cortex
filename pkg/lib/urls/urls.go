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

package urls

import (
	"net/url"
	"regexp"
	"strings"

	s "github.com/cortexlabs/cortex/pkg/lib/strings"
)

var (
	dns1035Regex = regexp.MustCompile(`^[a-z]([-a-z0-9]*[a-z0-9])?$`)
	dns1123Regex = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
)

func Parse(rawurl string) (*url.URL, error) {
	u, err := url.Parse(rawurl)
	if err != nil {
		return nil, ErrorInvalidURL(rawurl)
	}
	return u, nil
}

func Join(strs ...string) string {
	fullPath := ""
	for i, str := range strs {
		if i == 0 {
			fullPath = str
		} else {
			fullPath = s.EnsureSuffix(fullPath, "/")
			fullPath = fullPath + strings.TrimPrefix(str, "/")
		}
	}
	return fullPath
}

func CheckDNS1035(s string) error {
	if !dns1035Regex.MatchString(s) {
		return ErrorDNS1035(s)
	}
	return nil
}

func CheckDNS1123(s string) error {
	if !dns1123Regex.MatchString(s) {
		return ErrorDNS1123(s)
	}
	return nil
}
