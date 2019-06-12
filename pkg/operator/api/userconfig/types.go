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

package userconfig

import (
	"strings"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/cast"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
)

func DataTypeStrsOr(dataTypes []interface{}) string {
	dataTypeStrs := make([]string, len(dataTypes))
	for i, dataType := range dataTypes {
		dataTypeStrs[i] = DataTypeStr(dataType)
	}
	return s.StrsOr(dataTypeStrs)
}

func DataTypeStr(dataType interface{}) string {
	dataTypeStr := s.ObjFlat(flattenTypeSchema(dataType))
	matches := consts.TypeStrRegex.FindAllString(dataTypeStr, -1)
	for _, match := range matches {
		trimmed := s.TrimPrefixAndSuffix(match, `"`)
		dataTypeStr = strings.Replace(dataTypeStr, match, trimmed, -1)
	}
	return dataTypeStr
}

func DataTypeUserStr(dataType interface{}) string {
	dataTypeStr := DataTypeStr(dataType)
	if consts.SingleTypeStrRegex.MatchString(dataTypeStr) {
		dataTypeStr = s.UserStr(dataTypeStr)
	}
	return dataTypeStr
}

// Remove cortex arg options from input schemas
func flattenTypeSchema(schema interface{}) interface{} {
	if inputSchema, ok := schema.(InputSchema); ok {
		return flattenTypeSchema(inputSchema.Type)
	}

	if inputSchemaPtr, ok := schema.(*InputSchema); ok {
		return flattenTypeSchema(inputSchemaPtr.Type)
	}

	if schemaSlice, ok := cast.InterfaceToInterfaceSlice(schema); ok {
		flattenedSlice := make([]interface{}, len(schemaSlice))
		for i := range schemaSlice {
			flattenedSlice[i] = flattenTypeSchema(schemaSlice[i])
		}
		return flattenedSlice
	}

	if schemaMap, ok := cast.InterfaceToInterfaceInterfaceMap(schema); ok {
		flattenedMap := make(map[interface{}]interface{}, len(schemaMap))
		for k, v := range schemaMap {
			flattenedMap[k] = flattenTypeSchema(v)
		}
		return flattenedMap
	}

	return schema
}
