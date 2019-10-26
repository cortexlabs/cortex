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

package table

import (
	"strings"

	s "github.com/cortexlabs/cortex/pkg/lib/strings"
)

type KV struct {
	K interface{}
	V interface{}
}

func AlignKeyValue(pairs []KV, delimiter string, numSpaces int) string {
	var maxLen int
	for _, pair := range pairs {
		keyLen := len(s.ObjFlatNoQuotes(pair.K))
		if keyLen > maxLen {
			maxLen = keyLen
		}
	}

	var b strings.Builder
	for _, pair := range pairs {
		keyStr := s.ObjFlatNoQuotes(pair.K)
		valStr := s.ObjFlatNoQuotes(pair.V)
		keyLen := len(keyStr)
		spaces := strings.Repeat(" ", maxLen-keyLen+numSpaces)
		b.WriteString(keyStr + delimiter + spaces + valStr + "\n")
	}

	return strings.TrimSpace(b.String())
}
