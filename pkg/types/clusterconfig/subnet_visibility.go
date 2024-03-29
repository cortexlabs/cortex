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

type SubnetVisibility int

const (
	UnknownSubnetVisibility SubnetVisibility = iota
	PublicSubnetVisibility
	PrivateSubnetVisibility
)

var _subnetVisibilities = []string{
	"unknown",
	"public",
	"private",
}

func SubnetVisibilityFromString(s string) SubnetVisibility {
	for i := 0; i < len(_subnetVisibilities); i++ {
		if s == _subnetVisibilities[i] {
			return SubnetVisibility(i)
		}
	}
	return UnknownSubnetVisibility
}

func SubnetVisibilityStrings() []string {
	return _subnetVisibilities[1:]
}

func (t SubnetVisibility) String() string {
	return _subnetVisibilities[t]
}

// MarshalText satisfies TextMarshaler
func (t SubnetVisibility) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// UnmarshalText satisfies TextUnmarshaler
func (t *SubnetVisibility) UnmarshalText(text []byte) error {
	enum := string(text)
	for i := 0; i < len(_subnetVisibilities); i++ {
		if enum == _subnetVisibilities[i] {
			*t = SubnetVisibility(i)
			return nil
		}
	}

	*t = UnknownSubnetVisibility
	return nil
}

// UnmarshalBinary satisfies BinaryUnmarshaler
// Needed for msgpack
func (t *SubnetVisibility) UnmarshalBinary(data []byte) error {
	return t.UnmarshalText(data)
}

// MarshalBinary satisfies BinaryMarshaler
func (t SubnetVisibility) MarshalBinary() ([]byte, error) {
	return []byte(t.String()), nil
}
