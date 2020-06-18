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

package strings

import (
	"strings"
	"unicode"

	"github.com/cortexlabs/cortex/pkg/lib/cast"
)

func ToTitle(str string) string {
	return strings.Title(strings.ToLower(str))
}

func EnsurePrefix(str string, prefix string) string {
	if prefix != "" && !strings.HasPrefix(str, prefix) {
		return prefix + str
	}
	return str
}

func EnsureSuffix(str string, suffix string) string {
	if suffix != "" && !strings.HasSuffix(str, suffix) {
		return str + suffix
	}
	return str
}

func EnsureBlankLineIfNotEmpty(str string) string {
	if str == "" {
		return str
	}
	if strings.HasSuffix(str, "\n\n") {
		return str
	}
	if strings.HasSuffix(str, "\n") {
		return str + "\n"
	}
	return str + "\n\n"
}

func TrimTrailingNewLines(str string) string {
	return strings.TrimRight(str, "\n")
}

func TrimTrailingWhitespace(str string) string {
	return strings.TrimRightFunc(str, unicode.IsSpace)
}

func EnsureSingleTrailingNewLine(str string) string {
	return strings.TrimRight(str, "\n") + "\n"
}

func HasPrefixAndSuffix(str string, substr string) bool {
	return strings.HasPrefix(str, substr) && strings.HasSuffix(str, substr)
}

func TrimPrefixAndSuffix(str string, substr string) string {
	return strings.TrimSuffix(strings.TrimPrefix(str, substr), substr)
}

// MaskString omits no more than half of the string
func MaskString(str string, numPlain int) string {
	if numPlain > len(str)/2 {
		numPlain = len(str) / 2
	}
	return strings.Repeat("*", len(str)-numPlain) + str[len(str)-numPlain:]
}

func LongestCommonPrefix(strs ...string) string {
	if len(strs) == 0 {
		return ""
	}

	prefix := strs[0]

	if len(strs) == 1 {
		return prefix
	}

	for _, str := range strs[1:] {
		if len(prefix) == 0 || len(str) == 0 {
			return ""
		}

		maxLen := len(prefix)
		if len(str) < maxLen {
			maxLen = len(str)
		}
		for i := 0; i < maxLen; i++ {
			if prefix[i] != str[i] {
				prefix = prefix[:i]
				break
			}
		}
	}

	return prefix
}

func MaxLen(strs ...string) int {
	if len(strs) == 0 {
		return 0
	}

	maxLen := len(strs[0])
	for _, str := range strs {
		if len(str) > maxLen {
			maxLen = len(str)
		}
	}

	return maxLen
}

func TrimPrefixIfPresentInAll(strs []string, prefix string) ([]string, bool) {
	if prefix == "" {
		return strs, false
	}
	trimmedStrs := make([]string, len(strs))
	for i, str := range strs {
		if !strings.HasPrefix(str, prefix) {
			return strs, false
		}
		trimmedStrs[i] = strings.TrimPrefix(str, prefix)
	}
	return trimmedStrs, true
}

func StrsOr(strs []string) string {
	return StrsSentence(strs, "or")
}

func StrsAnd(strs []string) string {
	return StrsSentence(strs, "and")
}

func UserStrsOr(vals interface{}) string {
	return StrsOr(UserStrs(vals))
}

func UserStrsAnd(vals interface{}) string {
	return StrsAnd(UserStrs(vals))
}

func StrsSentence(strs []string, lastJoinWord string) string {
	switch len(strs) {
	case 0:
		return ""
	case 1:
		return strs[0]
	case 2:
		return strings.Join(strs, " "+lastJoinWord+" ")
	default:
		lastIndex := len(strs) - 1
		return strings.Join(strs[:lastIndex], ", ") + ", " + lastJoinWord + " " + strs[lastIndex]
	}
}

func PluralS(str string, count interface{}) string {
	return PluralCustom(str, str+"s", count)
}

func PluralEs(str string, count interface{}) string {
	return PluralCustom(str, str+"es", count)
}

func PluralCustom(singular string, plural string, count interface{}) string {
	countInt, _ := cast.InterfaceToInt64(count)
	if countInt == 1 {
		return singular
	}
	return plural
}
