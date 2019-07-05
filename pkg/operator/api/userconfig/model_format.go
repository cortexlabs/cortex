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

type ModelFormat int

const (
	UnknownModelFormat ModelFormat = iota
	TensorFlowModelFormat
	ONNXModelFormat
)

var modelFormats = []string{
	"unknown",
	"tensorflow",
	"onnx",
}

func ModelFormatFromString(s string) ModelFormat {
	for i := 0; i < len(modelFormats); i++ {
		if s == modelFormats[i] {
			return ModelFormat(i)
		}
	}
	return UnknownModelFormat
}

func ModelFormatStrings() []string {
	return modelFormats[1:]
}

func (t ModelFormat) String() string {
	return modelFormats[t]
}

// MarshalText satisfies TextMarshaler
func (t ModelFormat) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// UnmarshalText satisfies TextUnmarshaler
func (t *ModelFormat) UnmarshalText(text []byte) error {
	enum := string(text)
	for i := 0; i < len(modelFormats); i++ {
		if enum == modelFormats[i] {
			*t = ModelFormat(i)
			return nil
		}
	}

	*t = UnknownModelFormat
	return nil
}

// UnmarshalBinary satisfies BinaryUnmarshaler
// Needed for msgpack
func (t *ModelFormat) UnmarshalBinary(data []byte) error {
	return t.UnmarshalText(data)
}

// MarshalBinary satisfies BinaryMarshaler
func (t ModelFormat) MarshalBinary() ([]byte, error) {
	return []byte(t.String()), nil
}
