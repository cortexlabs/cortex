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

package types

type ProviderType int

const (
	UnknownProviderType ProviderType = iota
	AWSProviderType
	GCPProviderType
)

var _providerTypes = []string{
	"unknown",
	"aws",
	"gcp",
}

var _ = [1]int{}[int(GCPProviderType)-(len(_providerTypes)-1)] // Ensure list length matches

func ProviderTypeFromString(s string) ProviderType {
	for i := 0; i < len(_providerTypes); i++ {
		if s == _providerTypes[i] {
			return ProviderType(i)
		}
	}
	return UnknownProviderType
}

func ProviderTypeStrings() []string {
	return _providerTypes[1:]
}

func ClusterProviderTypeStrings() []string {
	return _providerTypes[1:]
}

func (t ProviderType) String() string {
	return _providerTypes[t]
}

// MarshalText satisfies TextMarshaler
func (t ProviderType) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// UnmarshalText satisfies TextUnmarshaler
func (t *ProviderType) UnmarshalText(text []byte) error {
	enum := string(text)
	for i := 0; i < len(_providerTypes); i++ {
		if enum == _providerTypes[i] {
			*t = ProviderType(i)
			return nil
		}
	}

	*t = UnknownProviderType
	return nil
}

// UnmarshalBinary satisfies BinaryUnmarshaler
// Needed for msgpack
func (t *ProviderType) UnmarshalBinary(data []byte) error {
	return t.UnmarshalText(data)
}

// MarshalBinary satisfies BinaryMarshaler
func (t ProviderType) MarshalBinary() ([]byte, error) {
	return []byte(t.String()), nil
}
