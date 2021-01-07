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

type VolumeType int

const (
	UnknownVolumeType VolumeType = iota
	GP2VolumeType
	IO1VolumeType
	SC1VolumeType
	ST1VolumeType
)

var _availableVolumeTypes = []string{
	"unknown",
	"gp2",
	"io1",
	"sc1",
	"st1",
}

//VolumeTypeFromString turns string into StorageType
func VolumeTypeFromString(s string) VolumeType {
	for i := 0; i < len(_availableVolumeTypes); i++ {
		if s == _availableVolumeTypes[i] {
			return VolumeType(i)
		}
	}
	return UnknownVolumeType
}

func VolumeTypesStrings() []string {
	return _availableVolumeTypes[1:]
}

func (t VolumeType) String() string {
	return _availableVolumeTypes[t]
}

// MarshalText satisfies TextMarshaler
func (t VolumeType) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// UnmarshalText satisfies TextUnmarshaler
func (t *VolumeType) UnmarshalText(text []byte) error {
	enum := string(text)
	for i := 0; i < len(_availableVolumeTypes); i++ {
		if enum == _availableVolumeTypes[i] {
			*t = VolumeType(i)
			return nil
		}
	}

	*t = UnknownVolumeType
	return nil
}

// UnmarshalBinary satisfies BinaryUnmarshaler
// Needed for msgpack
func (t *VolumeType) UnmarshalBinary(data []byte) error {
	return t.UnmarshalText(data)
}

// MarshalBinary satisfies BinaryMarshaler
func (t VolumeType) MarshalBinary() ([]byte, error) {
	return []byte(t.String()), nil
}
