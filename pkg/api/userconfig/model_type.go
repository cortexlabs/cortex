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

type ModelType int

const (
	UnknownModelType ModelType = iota
	ClassificationModelType
	RegressionModelType
)

var modelTypes = []string{
	"unknown",
	"classification",
	"regression",
}

func ModelTypeFromString(s string) ModelType {
	for i := 0; i < len(modelTypes); i++ {
		if s == modelTypes[i] {
			return ModelType(i)
		}
	}
	return UnknownModelType
}

func ModelTypeStrings() []string {
	return modelTypes[1:]
}

func (t ModelType) String() string {
	return modelTypes[t]
}

// MarshalText satisfies TextMarshaler
func (t ModelType) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// UnmarshalText satisfies TextUnmarshaler
func (t *ModelType) UnmarshalText(text []byte) error {
	enum := string(text)
	for i := 0; i < len(modelTypes); i++ {
		if enum == modelTypes[i] {
			*t = ModelType(i)
			return nil
		}
	}

	*t = UnknownModelType
	return nil
}

// UnmarshalBinary satisfies BinaryUnmarshaler
// Needed for msgpack
func (t *ModelType) UnmarshalBinary(data []byte) error {
	return t.UnmarshalText(data)
}

// MarshalBinary satisfies BinaryMarshaler
func (t ModelType) MarshalBinary() ([]byte, error) {
	return []byte(t.String()), nil
}
