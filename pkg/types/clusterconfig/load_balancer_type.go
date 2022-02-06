/*
Copyright 2022 Cortex Labs, Inc.

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

type LoadBalancerType int

const (
	UnknownLoadBalancerType LoadBalancerType = iota
	NLBLoadBalancerType
	ELBLoadBalancerType
)

var _loadBalancerTypes = []string{
	"unknown",
	"nlb",
	"elb",
}

func LoadBalancerTypeFromString(s string) LoadBalancerType {
	for i := 0; i < len(_loadBalancerTypes); i++ {
		if s == _loadBalancerTypes[i] {
			return LoadBalancerType(i)
		}
	}
	return UnknownLoadBalancerType
}

func LoadBalancerTypeStrings() []string {
	return _loadBalancerTypes[1:]
}

func (t LoadBalancerType) String() string {
	return _loadBalancerTypes[t]
}

// MarshalText satisfies TextMarshaler
func (t LoadBalancerType) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// UnmarshalText satisfies TextUnmarshaler
func (t *LoadBalancerType) UnmarshalText(text []byte) error {
	enum := string(text)
	for i := 0; i < len(_loadBalancerTypes); i++ {
		if enum == _loadBalancerTypes[i] {
			*t = LoadBalancerType(i)
			return nil
		}
	}

	*t = UnknownLoadBalancerType
	return nil
}

// UnmarshalBinary satisfies BinaryUnmarshaler
// Needed for msgpack
func (t *LoadBalancerType) UnmarshalBinary(data []byte) error {
	return t.UnmarshalText(data)
}

// MarshalBinary satisfies BinaryMarshaler
func (t LoadBalancerType) MarshalBinary() ([]byte, error) {
	return []byte(t.String()), nil
}
