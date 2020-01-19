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
	"fmt"
	"strings"

	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
)

type KeyValuePairOpts struct {
	Delimiter *string // default: ":"
	NumSpaces *int    // default: 1
}

type KeyValuePairs struct {
	kvs []kv
}

type kv struct {
	k interface{}
	v interface{}
}

func (kvs *KeyValuePairs) Add(key interface{}, value interface{}) {
	kvs.kvs = append(kvs.kvs, kv{k: key, v: value})
}

func (kvs *KeyValuePairs) AddAll(kvs2 KeyValuePairs) {
	for _, pair := range kvs2.kvs {
		kvs.Add(pair.k, pair.v)
	}
}

func (kvs KeyValuePairs) String(options ...*KeyValuePairOpts) string {
	opts := mergeOptions(options...)

	var maxLen int
	for _, pair := range kvs.kvs {
		keyLen := len(s.ObjFlatNoQuotes(pair.k))
		if keyLen > maxLen {
			maxLen = keyLen
		}
	}

	var b strings.Builder
	for _, pair := range kvs.kvs {
		keyStr := s.ObjFlatNoQuotes(pair.k)
		valStr := s.ObjFlatNoQuotes(pair.v)
		keyLen := len(keyStr)
		spaces := strings.Repeat(" ", maxLen-keyLen+*opts.NumSpaces)
		b.WriteString(keyStr + *opts.Delimiter + spaces + valStr + "\n")
	}

	return b.String()
}

func (kvs KeyValuePairs) Print(options ...*KeyValuePairOpts) {
	fmt.Println(kvs.String(options...))
}

func mergeOptions(options ...*KeyValuePairOpts) KeyValuePairOpts {
	mergedOpts := KeyValuePairOpts{}

	for _, opt := range options {
		if opt != nil && opt.Delimiter != nil {
			mergedOpts.Delimiter = opt.Delimiter
		}
		if opt != nil && opt.NumSpaces != nil {
			mergedOpts.NumSpaces = opt.NumSpaces
		}
	}

	if mergedOpts.Delimiter == nil {
		mergedOpts.Delimiter = pointer.String(":")
	}
	if mergedOpts.NumSpaces == nil {
		mergedOpts.NumSpaces = pointer.Int(1)
	}

	return mergedOpts
}
