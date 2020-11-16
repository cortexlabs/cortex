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

package flags

import "github.com/cortexlabs/cortex/pkg/types"

type CloudProviderType int

const (
	UnknownCloudProviderType CloudProviderType = iota
	AWSCloudProviderType
	GCPCloudProviderType
)

// subset of types.ProviderType
var _cloudProviderType = []string{
	"unknown",
	"aws",
	"gcp",
}

var _ = [1]int{}[int(GCPCloudProviderType)-(len(_cloudProviderType)-1)] // Ensure list length matches

func CloudProviderTypeFromString(s string) CloudProviderType {
	for i := 0; i < len(_cloudProviderType); i++ {
		if s == _cloudProviderType[i] {
			return CloudProviderType(i)
		}
	}
	return UnknownCloudProviderType
}

func CloudProviderTypeStrings() []string {
	return _cloudProviderType[1:]
}

func (t CloudProviderType) String() string {
	return _cloudProviderType[t]
}

func (t CloudProviderType) ToProvider() types.ProviderType {
	return types.ProviderTypeFromString(_cloudProviderType[t])
}

// MarshalText satisfies TextMarshaler
func (t CloudProviderType) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// UnmarshalText satisfies TextUnmarshaler
func (t *CloudProviderType) UnmarshalText(text []byte) error {
	enum := string(text)
	for i := 0; i < len(_cloudProviderType); i++ {
		if enum == _cloudProviderType[i] {
			*t = CloudProviderType(i)
			return nil
		}
	}

	*t = UnknownCloudProviderType
	return nil
}

// UnmarshalBinary satisfies BinaryUnmarshaler
// Needed for msgpack
func (t *CloudProviderType) UnmarshalBinary(data []byte) error {
	return t.UnmarshalText(data)
}

// MarshalBinary satisfies BinaryMarshaler
func (t CloudProviderType) MarshalBinary() ([]byte, error) {
	return []byte(t.String()), nil
}

// Satisfies Cobra interface for parsing flags to custom types
func (t *CloudProviderType) Set(value string) error {
	provider := CloudProviderTypeFromString(value)
	if provider == UnknownCloudProviderType {
		return ErrorInvalidCloudProviderType(value)
	}
	*t = provider
	return nil
}

// Satisfies Cobra interface for parsing flags to custom types
func (t CloudProviderType) Type() string {
	return "string"
}
