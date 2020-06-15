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

package userconfig

type APIGatewayType int

const (
	UnknownAPIGatewayType APIGatewayType = iota
	PublicAPIGatewayType
	NoneAPIGatewayType
)

var _apiGatewayTypes = []string{
	"unknown",
	"public",
	"none",
}

func APIGatewayTypeFromString(s string) APIGatewayType {
	for i := 0; i < len(_apiGatewayTypes); i++ {
		if s == _apiGatewayTypes[i] {
			return APIGatewayType(i)
		}
	}
	return UnknownAPIGatewayType
}

func APIGatewayTypeStrings() []string {
	return _apiGatewayTypes[1:]
}

func (t APIGatewayType) String() string {
	return _apiGatewayTypes[t]
}

// MarshalText satisfies TextMarshaler
func (t APIGatewayType) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// UnmarshalText satisfies TextUnmarshaler
func (t *APIGatewayType) UnmarshalText(text []byte) error {
	enum := string(text)
	for i := 0; i < len(_apiGatewayTypes); i++ {
		if enum == _apiGatewayTypes[i] {
			*t = APIGatewayType(i)
			return nil
		}
	}

	*t = UnknownAPIGatewayType
	return nil
}

// UnmarshalBinary satisfies BinaryUnmarshaler
// Needed for msgpack
func (t *APIGatewayType) UnmarshalBinary(data []byte) error {
	return t.UnmarshalText(data)
}

// MarshalBinary satisfies BinaryMarshaler
func (t APIGatewayType) MarshalBinary() ([]byte, error) {
	return []byte(t.String()), nil
}
