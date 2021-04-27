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

type HandlerType int

const (
	UnknownHandlerType HandlerType = iota
	PythonHandlerType
	TensorFlowHandlerType
)

var _handlerTypes = []string{
	"unknown",
	"python",
	"tensorflow",
}

var _casedHandlerTypes = []string{
	"unknown",
	"Python",
	"TensorFlow",
}

func HandlerTypeFromString(s string) HandlerType {
	for i := 0; i < len(_handlerTypes); i++ {
		if s == _handlerTypes[i] {
			return HandlerType(i)
		}
	}
	return UnknownHandlerType
}

func HandlerTypeStrings() []string {
	return _handlerTypes[1:]
}

func (t HandlerType) String() string {
	return _handlerTypes[t]
}

func (t HandlerType) CasedString() string {
	return _casedHandlerTypes[t]
}

// MarshalText satisfies TextMarshaler
func (t HandlerType) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// UnmarshalText satisfies TextUnmarshaler
func (t *HandlerType) UnmarshalText(text []byte) error {
	enum := string(text)
	for i := 0; i < len(_handlerTypes); i++ {
		if enum == _handlerTypes[i] {
			*t = HandlerType(i)
			return nil
		}
	}

	*t = UnknownHandlerType
	return nil
}

// UnmarshalBinary satisfies BinaryUnmarshaler
// Needed for msgpack
func (t *HandlerType) UnmarshalBinary(data []byte) error {
	return t.UnmarshalText(data)
}

// MarshalBinary satisfies BinaryMarshaler
func (t HandlerType) MarshalBinary() ([]byte, error) {
	return []byte(t.String()), nil
}
