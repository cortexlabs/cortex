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

import "strings"

type FeatureType int
type FeatureTypes []FeatureType

const (
	UnknownFeatureType FeatureType = iota
	IntegerFeatureType
	FloatFeatureType
	StringFeatureType
	IntegerListFeatureType
	FloatListFeatureType
	StringListFeatureType
)

var featureTypes = []string{
	"unknown",
	"INT_FEATURE",
	"FLOAT_FEATURE",
	"STRING_FEATURE",
	"INT_LIST_FEATURE",
	"FLOAT_LIST_FEATURE",
	"STRING_LIST_FEATURE",
}

func FeatureTypeFromString(s string) FeatureType {
	for i := 0; i < len(featureTypes); i++ {
		if s == featureTypes[i] {
			return FeatureType(i)
		}
	}
	return UnknownFeatureType
}

func FeatureTypeStrings() []string {
	return featureTypes[1:]
}

func (t FeatureType) String() string {
	return featureTypes[t]
}

// MarshalText satisfies TextMarshaler
func (t FeatureType) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// UnmarshalText satisfies TextUnmarshaler
func (t *FeatureType) UnmarshalText(text []byte) error {
	enum := string(text)
	for i := 0; i < len(featureTypes); i++ {
		if enum == featureTypes[i] {
			*t = FeatureType(i)
			return nil
		}
	}

	*t = UnknownFeatureType
	return nil
}

// UnmarshalBinary satisfies BinaryUnmarshaler
// Needed for msgpack
func (t *FeatureType) UnmarshalBinary(data []byte) error {
	return t.UnmarshalText(data)
}

// MarshalBinary satisfies BinaryMarshaler
func (t FeatureType) MarshalBinary() ([]byte, error) {
	return []byte(t.String()), nil
}

func (ts FeatureTypes) StringList() []string {
	strs := make([]string, len(ts))
	for i, t := range ts {
		strs[i] = t.String()
	}
	return strs
}

func (ts FeatureTypes) String() string {
	return strings.Join(ts.StringList(), ", ")
}
