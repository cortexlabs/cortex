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

package context

import (
	"sort"
	"strings"

	s "github.com/cortexlabs/cortex/pkg/api/strings"
	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/utils/util"
)

func DataTypeID(dataType interface{}) string {
	dataTypeStr := s.Obj(dataType)
	matches := consts.CompoundTypeStrRegex.FindAllString(dataTypeStr, -1)
	for _, match := range matches {
		trimmed := s.TrimPrefixAndSuffix(match, `"`)
		parts := strings.Split(trimmed, "|")
		sort.Strings(parts)
		replacement := strings.Join(parts, "|")
		dataTypeStr = strings.Replace(dataTypeStr, match, replacement, -1)
	}
	return util.HashStr(dataTypeStr)
}
