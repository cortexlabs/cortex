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

package userconfig

type EnvironmentDataType int

const (
  UnknownEnvironmentDataType EnvironmentDataType = iota
  CSVEnvironmentDataType
  ParquetEnvironmentDataType
)

var environmentDataTypes = []string{
  "unknown",
  "csv",
  "parquet",
}

func EnvironmentDataTypeFromString(s string) EnvironmentDataType {
  for i := 0; i < len(environmentDataTypes); i++ {
    if s == environmentDataTypes[i] {
      return EnvironmentDataType(i)
    }
  }
  return UnknownEnvironmentDataType
}

func EnvironmentDataTypeStrings() []string {
  return environmentDataTypes[1:]
}

func (t EnvironmentDataType) String() string {
  return environmentDataTypes[t]
}

// MarshalText satisfies TextMarshaler
func (t EnvironmentDataType) MarshalText() ([]byte, error) {
  return []byte(t.String()), nil
}

// UnmarshalText satisfies TextUnmarshaler
func (t *EnvironmentDataType) UnmarshalText(text []byte) error {
  enum := string(text)
  for i := 0; i < len(environmentDataTypes); i++ {
    if enum == environmentDataTypes[i] {
      *t = EnvironmentDataType(i)
      return nil
    }
  }

  *t = UnknownEnvironmentDataType
  return nil
}

// UnmarshalBinary satisfies BinaryUnmarshaler
// Needed for msgpack
func (t *EnvironmentDataType) UnmarshalBinary(data []byte) error {
  return t.UnmarshalText(data)
}

// MarshalBinary satisfies BinaryMarshaler
func (t EnvironmentDataType) MarshalBinary() ([]byte, error) {
  return []byte(t.String()), nil
}
