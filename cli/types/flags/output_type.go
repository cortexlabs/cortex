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

package flags

type OutputType int

const (
	UnknownOutputType OutputType = iota
	PrettyOutputType
	JSONOutputType
)

var _outputTypes = []string{
	"unknown",
	"pretty",
	"json",
}

func OutputTypeFromString(s string) OutputType {
	for i := 0; i < len(_outputTypes); i++ {
		if s == _outputTypes[i] {
			return OutputType(i)
		}
	}
	return UnknownOutputType
}

func UserOutputTypeStrings() []string {
	return _outputTypes[2:]
}

func OutputTypeStrings() []string {
	return _outputTypes[1:]
}

func (t OutputType) String() string {
	return _outputTypes[t]
}

// MarshalText satisfies TextMarshaler
func (t OutputType) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// UnmarshalText satisfies TextUnmarshaler
func (t *OutputType) UnmarshalText(text []byte) error {
	enum := string(text)
	for i := 0; i < len(_outputTypes); i++ {
		if enum == _outputTypes[i] {
			*t = OutputType(i)
			return nil
		}
	}

	*t = UnknownOutputType
	return nil
}

// UnmarshalBinary satisfies BinaryUnmarshaler
// Needed for msgpack
func (t *OutputType) UnmarshalBinary(data []byte) error {
	return t.UnmarshalText(data)
}

// MarshalBinary satisfies BinaryMarshaler
func (t OutputType) MarshalBinary() ([]byte, error) {
	return []byte(t.String()), nil
}

func (t *OutputType) Set(value string) error {
	output := OutputTypeFromString(value)
	if output == UnknownOutputType {
		return ErrorInvalidOutputType(value)
	}
	*t = output
	return nil
}

func (t OutputType) Type() string {
	return "string"
}
