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

package cloud

import "fmt"

type ProviderType int

const (
	UnknownProviderType ProviderType = iota
	AWSProviderType
	LocalProviderType
)

var providerTypes = []string{
	"unknown",
	"aws",
	"local",
}

func ProviderTypeFromString(s string) ProviderType {
	for i := 0; i < len(providerTypes); i++ {
		if s == providerTypes[i] {
			return ProviderType(i)
		}
	}
	return UnknownProviderType
}

func ProviderTypeStrings() []string {
	return providerTypes[1:]
}

func (p ProviderType) String() string {
	return providerTypes[p]
}

// MarshalText satisfies TextMarshaler
func (p ProviderType) MarshalText() ([]byte, error) {
	return []byte(p.String()), nil
}

// UnmarshalText satisfies TextUnmarshaler
func (p *ProviderType) UnmarshalText(text []byte) error {
	fmt.Println(string(text))
	enum := string(text)
	for i := 0; i < len(providerTypes); i++ {
		if enum == providerTypes[i] {
			*p = ProviderType(i)
			return nil
		}
	}

	*p = UnknownProviderType
	return nil
}

func ParserFunc(str string) (interface{}, error) {
	providerType := ProviderTypeFromString(str)
	if providerType == UnknownProviderType {
		return providerType, ErrorUnsupportedProviderType(str)
	}
	return providerType, nil
}
