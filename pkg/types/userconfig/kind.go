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

type Kind int

const (
	UnknownKind Kind = iota
	RealtimeAPIKind
	BatchAPIKind
	TrafficSplitterKind
	TaskAPIKind
	AsyncAPIKind
)

var _kinds = []string{
	"unknown",
	"RealtimeAPI",
	"BatchAPI",
	"TrafficSplitter",
	"TaskAPI",
	"AsyncAPI",
}

func KindFromString(s string) Kind {
	for i := 0; i < len(_kinds); i++ {
		if s == _kinds[i] {
			return Kind(i)
		}
	}
	return UnknownKind
}

func KindStrings() []string {
	return _kinds[1:]
}

func (t Kind) String() string {
	return _kinds[t]
}

// MarshalText satisfies TextMarshaler
func (t Kind) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// UnmarshalText satisfies TextUnmarshaler
func (t *Kind) UnmarshalText(text []byte) error {
	enum := string(text)
	for i := 0; i < len(_kinds); i++ {
		if enum == _kinds[i] {
			*t = Kind(i)
			return nil
		}
	}

	*t = UnknownKind
	return nil
}

// UnmarshalBinary satisfies BinaryUnmarshaler
// Needed for msgpack
func (t *Kind) UnmarshalBinary(data []byte) error {
	return t.UnmarshalText(data)
}

// MarshalBinary satisfies BinaryMarshaler
func (t Kind) MarshalBinary() ([]byte, error) {
	return []byte(t.String()), nil
}
