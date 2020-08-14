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

package clusterconfig

type APIGatewaySetting int

const (
	UnknownAPIGatewaySetting APIGatewaySetting = iota
	EnabledAPIGatewaySetting
	DisabledAPIGatewaySetting
)

var _apiGatewaySettings = []string{
	"unknown",
	"enabled",
	"disabled",
}

func APIGatewaySettingFromString(s string) APIGatewaySetting {
	for i := 0; i < len(_apiGatewaySettings); i++ {
		if s == _apiGatewaySettings[i] {
			return APIGatewaySetting(i)
		}
	}
	return UnknownAPIGatewaySetting
}

func APIGatewaySettingStrings() []string {
	return _apiGatewaySettings[1:]
}

func (t APIGatewaySetting) String() string {
	return _apiGatewaySettings[t]
}

// MarshalText satisfies TextMarshaler
func (t APIGatewaySetting) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// UnmarshalText satisfies TextUnmarshaler
func (t *APIGatewaySetting) UnmarshalText(text []byte) error {
	enum := string(text)
	for i := 0; i < len(_apiGatewaySettings); i++ {
		if enum == _apiGatewaySettings[i] {
			*t = APIGatewaySetting(i)
			return nil
		}
	}

	*t = UnknownAPIGatewaySetting
	return nil
}

// UnmarshalBinary satisfies BinaryUnmarshaler
// Needed for msgpack
func (t *APIGatewaySetting) UnmarshalBinary(data []byte) error {
	return t.UnmarshalText(data)
}

// MarshalBinary satisfies BinaryMarshaler
func (t APIGatewaySetting) MarshalBinary() ([]byte, error) {
	return []byte(t.String()), nil
}
