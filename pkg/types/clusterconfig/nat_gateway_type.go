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

type NATGateway int

const (
	UnknownNATGateway NATGateway = iota
	NoneNATGateway
	SingleNATGateway
	HighlyAvailableNATGateway
)

var _natGateways = []string{
	"unknown",
	"none",
	"single",
	"highly_available",
}

func NATGatewayFromString(s string) NATGateway {
	for i := 0; i < len(_natGateways); i++ {
		if s == _natGateways[i] {
			return NATGateway(i)
		}
	}
	return UnknownNATGateway
}

func NATGatewayStrings() []string {
	return _natGateways[1:]
}

func (t NATGateway) String() string {
	return _natGateways[t]
}

// MarshalText satisfies TextMarshaler
func (t NATGateway) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// UnmarshalText satisfies TextUnmarshaler
func (t *NATGateway) UnmarshalText(text []byte) error {
	enum := string(text)
	for i := 0; i < len(_natGateways); i++ {
		if enum == _natGateways[i] {
			*t = NATGateway(i)
			return nil
		}
	}

	*t = UnknownNATGateway
	return nil
}

// UnmarshalBinary satisfies BinaryUnmarshaler
// Needed for msgpack
func (t *NATGateway) UnmarshalBinary(data []byte) error {
	return t.UnmarshalText(data)
}

// MarshalBinary satisfies BinaryMarshaler
func (t NATGateway) MarshalBinary() ([]byte, error) {
	return []byte(t.String()), nil
}
