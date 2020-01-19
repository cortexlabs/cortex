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

package regex

import (
	"regexp"
)

func MatchAnyRegex(s string, regexes []*regexp.Regexp) bool {
	for _, regex := range regexes {
		if regex.MatchString(s) {
			return true
		}
	}
	return false
}

var _alphaNumericDashDotUnderscoreRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-\.]+$`)

func IsAlphaNumericDashDotUnderscore(s string) bool {
	return _alphaNumericDashDotUnderscoreRegex.MatchString(s)
}

var _alphaNumericDashUnderscoreRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-]+$`)

func IsAlphaNumericDashUnderscore(s string) bool {
	return _alphaNumericDashUnderscoreRegex.MatchString(s)
}
