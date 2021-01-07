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

package table

import (
	"fmt"
	"strings"

	"github.com/cortexlabs/cortex/pkg/lib/console"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
)

type KeyValuePairOpts struct {
	Delimiter     *string // default: ":"
	NumSpaces     *int    // default: 1
	RightJustify  *bool   // default: false
	BoldFirstLine *bool   // default: false
	BoldKeys      *bool   // default: false
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
	for i, pair := range kvs.kvs {
		keyStr := s.ObjFlatNoQuotes(pair.k)
		keyLen := len(keyStr)

		if *opts.BoldKeys {
			keyStr = console.Bold(keyStr)
		}

		valStr := s.ObjFlatNoQuotes(pair.v)
		var str string
		if *opts.RightJustify {
			alignmentSpaces := strings.Repeat(" ", maxLen-keyLen)
			delimiterSpaces := strings.Repeat(" ", *opts.NumSpaces)
			str = alignmentSpaces + keyStr + *opts.Delimiter + delimiterSpaces + valStr + "\n"
		} else {
			spaces := strings.Repeat(" ", maxLen-keyLen+*opts.NumSpaces)
			str = keyStr + *opts.Delimiter + spaces + valStr + "\n"
		}

		if *opts.BoldFirstLine && i == 0 {
			str = console.Bold(str)
		}

		b.WriteString(str)
	}

	return b.String()
}

func (kvs KeyValuePairs) Print(options ...*KeyValuePairOpts) {
	fmt.Print(kvs.String(options...))
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
		if opt != nil && opt.RightJustify != nil {
			mergedOpts.RightJustify = opt.RightJustify
		}
		if opt != nil && opt.BoldFirstLine != nil {
			mergedOpts.BoldFirstLine = opt.BoldFirstLine
		}
		if opt != nil && opt.BoldKeys != nil {
			mergedOpts.BoldKeys = opt.BoldKeys
		}
	}

	if mergedOpts.Delimiter == nil {
		mergedOpts.Delimiter = pointer.String(":")
	}
	if mergedOpts.NumSpaces == nil {
		mergedOpts.NumSpaces = pointer.Int(1)
	}
	if mergedOpts.RightJustify == nil {
		mergedOpts.RightJustify = pointer.Bool(false)
	}
	if mergedOpts.BoldFirstLine == nil {
		mergedOpts.BoldFirstLine = pointer.Bool(false)
	}
	if mergedOpts.BoldKeys == nil {
		mergedOpts.BoldKeys = pointer.Bool(false)
	}

	return mergedOpts
}
