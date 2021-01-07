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

package clusterconfig

type LoadBalancerScheme int

const (
	UnknownLoadBalancerScheme LoadBalancerScheme = iota
	InternetFacingLoadBalancerScheme
	InternalLoadBalancerScheme
)

var _loadBalancerSchemes = []string{
	"unknown",
	"internet-facing",
	"internal",
}

func LoadBalancerSchemeFromString(s string) LoadBalancerScheme {
	for i := 0; i < len(_loadBalancerSchemes); i++ {
		if s == _loadBalancerSchemes[i] {
			return LoadBalancerScheme(i)
		}
	}
	return UnknownLoadBalancerScheme
}

func LoadBalancerSchemeStrings() []string {
	return _loadBalancerSchemes[1:]
}

func (t LoadBalancerScheme) String() string {
	return _loadBalancerSchemes[t]
}

// MarshalText satisfies TextMarshaler
func (t LoadBalancerScheme) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// UnmarshalText satisfies TextUnmarshaler
func (t *LoadBalancerScheme) UnmarshalText(text []byte) error {
	enum := string(text)
	for i := 0; i < len(_loadBalancerSchemes); i++ {
		if enum == _loadBalancerSchemes[i] {
			*t = LoadBalancerScheme(i)
			return nil
		}
	}

	*t = UnknownLoadBalancerScheme
	return nil
}

// UnmarshalBinary satisfies BinaryUnmarshaler
// Needed for msgpack
func (t *LoadBalancerScheme) UnmarshalBinary(data []byte) error {
	return t.UnmarshalText(data)
}

// MarshalBinary satisfies BinaryMarshaler
func (t LoadBalancerScheme) MarshalBinary() ([]byte, error) {
	return []byte(t.String()), nil
}
