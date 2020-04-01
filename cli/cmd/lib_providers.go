/*
Copyright 2020 Cortex Labs, Inc.

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

package cmd

type Provider int

const (
	UnknownProvider Provider = iota
	Local
	AWS
)

var _providers = []string{
	"unknown",
	"local",
	"aws",
}

var _ = [1]int{}[int(AWS)-(len(_providers)-1)] // Ensure list length matches

func ProviderFromString(s string) Provider {
	for i := 0; i < len(_providers); i++ {
		if s == _providers[i] {
			return Provider(i)
		}
	}
	return UnknownProvider
}

func ProviderStrings() []string {
	return _providers[1:]
}

func (t Provider) String() string {
	return _providers[t]
}

// MarshalText satisfies TextMarshaler
func (t Provider) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// UnmarshalText satisfies TextUnmarshaler
func (t *Provider) UnmarshalText(text []byte) error {
	enum := string(text)
	for i := 0; i < len(_providers); i++ {
		if enum == _providers[i] {
			*t = Provider(i)
			return nil
		}
	}

	*t = UnknownProvider
	return nil
}

// UnmarshalBinary satisfies BinaryUnmarshaler
// Needed for msgpack
func (t *Provider) UnmarshalBinary(data []byte) error {
	return t.UnmarshalText(data)
}

// MarshalBinary satisfies BinaryMarshaler
func (t Provider) MarshalBinary() ([]byte, error) {
	return []byte(t.String()), nil
}
