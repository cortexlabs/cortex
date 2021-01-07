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

package configreader

type TypePlaceholder struct {
	Type string `json:"type"`
}

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

var PrimTypeScalars = []PrimitiveType{PrimTypeInt, PrimTypeFloat, PrimTypeString, PrimTypeBool}

func (ts PrimitiveTypes) StringList() []string {
	strs := make([]string, len(ts))
	for i, t := range ts {
		strs[i] = string(t)
	}
	return strs
}
