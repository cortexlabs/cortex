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

package status

// MarshalText satisfies TextMarshaler
func (code Code) MarshalText() ([]byte, error) {
	return []byte(code.String()), nil
}

// UnmarshalText satisfies TextUnmarshaler
func (code *Code) UnmarshalText(text []byte) error {
	enum := string(text)
	for i := 0; i < len(codes); i++ {
		if enum == codes[i] {
			*code = Code(i)
			return nil
		}
	}

	*code = Unknown
	return nil
}

// UnmarshalBinary satisfies BinaryUnmarshaler
// Needed for msgpack
func (code *Code) UnmarshalBinary(data []byte) error {
	return code.UnmarshalText(data)
}

// MarshalBinary satisfies BinaryMarshaler
func (code Code) MarshalBinary() ([]byte, error) {
	return []byte(code.String()), nil
}
