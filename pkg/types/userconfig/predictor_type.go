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

package userconfig

type PredictorType int

const (
	UnknownPredictorType PredictorType = iota
	PythonPredictorType
	TensorFlowPredictorType
	ONNXPredictorType
)

var _predictorTypes = []string{
	"unknown",
	"python",
	"tensorflow",
	"onnx",
}

var _casedPredictorTypes = []string{
	"unknown",
	"Python",
	"TensorFlow",
	"ONNX",
}

func PredictorTypeFromString(s string) PredictorType {
	for i := 0; i < len(_predictorTypes); i++ {
		if s == _predictorTypes[i] {
			return PredictorType(i)
		}
	}
	return UnknownPredictorType
}

func PredictorTypeStrings() []string {
	return _predictorTypes[1:]
}

func (t PredictorType) String() string {
	return _predictorTypes[t]
}

func (t PredictorType) CasedString() string {
	return _casedPredictorTypes[t]
}

// MarshalText satisfies TextMarshaler
func (t PredictorType) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// UnmarshalText satisfies TextUnmarshaler
func (t *PredictorType) UnmarshalText(text []byte) error {
	enum := string(text)
	for i := 0; i < len(_predictorTypes); i++ {
		if enum == _predictorTypes[i] {
			*t = PredictorType(i)
			return nil
		}
	}

	*t = UnknownPredictorType
	return nil
}

// UnmarshalBinary satisfies BinaryUnmarshaler
// Needed for msgpack
func (t *PredictorType) UnmarshalBinary(data []byte) error {
	return t.UnmarshalText(data)
}

// MarshalBinary satisfies BinaryMarshaler
func (t PredictorType) MarshalBinary() ([]byte, error) {
	return []byte(t.String()), nil
}
