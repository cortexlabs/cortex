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

var _leadingWhitespaceRegex = regexp.MustCompile(`^\s+`)

func HasLeadingWhitespace(s string) bool {
	return _leadingWhitespaceRegex.MatchString(s)
}

var _trailingWhitespaceRegex = regexp.MustCompile(`\s+$`)

func HasTrailingWhitespace(s string) bool {
	return _trailingWhitespaceRegex.MatchString(s)
}

// letters, numbers, spaces representable in UTF-8, and the following characters: _ . : / + - @
// = is not supported because it doesn't propagate to the NLB correctly (via the k8s service annotation)
var _awsTagRegex = regexp.MustCompile(`^[\sa-zA-Z0-9_\-\.:/+@]+$`)

func IsValidAWSTag(s string) bool {
	return _awsTagRegex.MatchString(s)
}

var _alphaNumericDashDotUnderscoreRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-\.]+$`)

func IsAlphaNumericDashDotUnderscore(s string) bool {
	return _alphaNumericDashDotUnderscoreRegex.MatchString(s)
}

var _alphaNumericDashUnderscoreRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-]+$`)

func IsAlphaNumericDashUnderscore(s string) bool {
	return _alphaNumericDashUnderscoreRegex.MatchString(s)
}

// used the evaluated form of
// https://github.com/docker/distribution/blob/3150937b9f2b1b5b096b2634d0e7c44d4a0f89fb/reference/regexp.go#L68-L70
var _dockerValidImage = regexp.MustCompile(
	`^((?:(?:[a-z0-9]|[a-z0-9][a-z0-9-]*[a-z0-9])` +
		`(?:(?:\.(?:[a-z0-9]|[a-z0-9][a-z0-9-]*[a-z0-9]))+)` +
		`?(?::[0-9]+)?/)?[a-z0-9]` +
		`+(?:(?:(?:[._]|__|[-]*)[a-z0-9]+)+)` +
		`?(?:(?:/[a-z0-9]+(?:(?:(?:[._]|__|[-]*)[a-z0-9]+)+)?)+)?)` +
		`(?::([\w][\w.-]{0,127}))` +
		`?(?:@([A-Za-z][A-Za-z0-9]*(?:[-_+.][A-Za-z][A-Za-z0-9]*)*[:][[:xdigit:]]{32,}))?$`,
)

func IsValidDockerImage(s string) bool {
	return _dockerValidImage.MatchString(s)
}

var _ecrPattern = regexp.MustCompile(
	`(^[a-zA-Z0-9][a-zA-Z0-9-_]*)\.dkr\.ecr(\-fips)?\.([a-zA-Z0-9][a-zA-Z0-9-_]*)\.amazonaws\.com(\.cn)?`,
)

func IsValidECRURL(s string) bool {
	return _ecrPattern.MatchString(s)
}
