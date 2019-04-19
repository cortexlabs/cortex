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

package strings

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/cortexlabs/cortex/pkg/consts"
)

type PrimitiveType string
type PrimitiveTypes []PrimitiveType

var (
	PrimTypeInt        PrimitiveType = "integer"
	PrimTypeIntList    PrimitiveType = "integer list"
	PrimTypeFloat      PrimitiveType = "float"
	PrimTypeFloatList  PrimitiveType = "float list"
	PrimTypeString     PrimitiveType = "string"
	PrimTypeStringList PrimitiveType = "string list"
	PrimTypeBool       PrimitiveType = "boolean"
	PrimTypeBoolList   PrimitiveType = "boolean list"

	PrimTypeMap     PrimitiveType = "map"
	PrimTypeMapList PrimitiveType = "list of maps"
	PrimTypeList    PrimitiveType = "list"

	PrimTypeStringToStringMap PrimitiveType = "map of strings to strings"
)

func (ts PrimitiveTypes) StringList() []string {
	strs := make([]string, len(ts))
	for i, t := range ts {
		strs[i] = string(t)
	}
	return strs
}

func EnvVar(envVarName string) string {
	return fmt.Sprintf("environment variable \"%s\"", envVarName)
}

func DataTypeStrsOr(dataTypes []interface{}) string {
	dataTypeStrs := make([]string, len(dataTypes))
	for i, dataType := range dataTypes {
		dataTypeStrs[i] = DataTypeStr(dataType)
	}
	return StrsOr(dataTypeStrs)
}

func DataTypeStr(dataType interface{}) string {
	dataTypeStr := ObjFlat(dataType)
	matches := consts.TypeStrRegex.FindAllString(dataTypeStr, -1)
	for _, match := range matches {
		trimmed := TrimPrefixAndSuffix(match, `"`)
		dataTypeStr = strings.Replace(dataTypeStr, match, trimmed, -1)
	}
	return dataTypeStr
}

var singleDataTypeRegexp = regexp.MustCompile(`^\w*\w$`)

func DataTypeUserStr(dataType interface{}) string {
	dataTypeStr := DataTypeStr(dataType)
	if singleDataTypeRegexp.MatchString(dataTypeStr) {
		dataTypeStr = UserStr(dataTypeStr)
	}
	return dataTypeStr
}
