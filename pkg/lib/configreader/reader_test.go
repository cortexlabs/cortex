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

package configreader

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/stretchr/testify/require"
)

type SimpleConfig struct {
	Key1 bool `json:"key1,omitempty"`
	Key2 bool `json:"key2"`
}

func TestSimple(t *testing.T) {
	structValidation := &StructValidation{
		StructFieldValidations: []*StructFieldValidation{
			{
				// Key:         "key1",
				StructField: "Key1",
				BoolValidation: &BoolValidation{
					Required: true,
				},
			},
			{
				// Key:         "key2",
				StructField: "Key2",
				BoolValidation: &BoolValidation{
					Default: true,
				},
			},
		},
		Required:     true,
		ShortCircuit: true,
	}

	configData := MustReadYAMLStr(
		`
    key1: true
    `)

	expected := &SimpleConfig{
		Key1: true,
		Key2: true,
	}

	testConfig(structValidation, configData, expected, t)
}

type NestedConfig struct {
	Key0 float64  `json:"key0"`
	Key1 *Nested1 `json:"key1"`
	Key2 *Nested2 `json:"key2"`
}
type Nested1 struct {
	Key11 int32 `json:"key11"`
}
type Nested2 struct {
	Key21 string   `json:"key21"`
	Key22 *Nested3 `json:"key22"`
}
type Nested3 struct {
	Key31 int `json:"key31"`
}

func TestNested(t *testing.T) {
	structValidation := &StructValidation{
		StructFieldValidations: []*StructFieldValidation{
			{
				// Key:               "key0",
				StructField:       "Key0",
				Float64Validation: &Float64Validation{},
			},
			{
				// Key:         "key1",
				StructField: "Key1",
				StructValidation: &StructValidation{
					StructFieldValidations: []*StructFieldValidation{
						{
							// Key:             "key11",
							StructField:     "Key11",
							Int32Validation: &Int32Validation{},
						},
					},
					Required:     true,
					ShortCircuit: true,
				},
			},
			{
				// Key:         "key2",
				StructField: "Key2",
				StructValidation: &StructValidation{
					StructFieldValidations: []*StructFieldValidation{
						{
							// Key:              "key21",
							StructField:      "Key21",
							StringValidation: &StringValidation{},
						},
						{
							// Key:         "key22",
							StructField: "Key22",
							StructValidation: &StructValidation{
								StructFieldValidations: []*StructFieldValidation{
									{
										// Key:           "key31",
										StructField:   "Key31",
										IntValidation: &IntValidation{},
									},
								},
								Required:     true,
								ShortCircuit: true,
							},
						},
					},
					Required:     true,
					ShortCircuit: true,
				},
			},
		},
		Required:     true,
		ShortCircuit: true,
	}

	configData := MustReadYAMLStr(
		`
    key0: 1.1
    key1:
      key11: 2
    key2:
      key21: test
      key22:
        key31: 0
    `)

	expected := &NestedConfig{
		Key0: 1.1,
		Key1: &Nested1{
			Key11: 2,
		},
		Key2: &Nested2{
			Key21: "test",
			Key22: &Nested3{
				Key31: 0,
			},
		},
	}

	testConfig(structValidation, configData, expected, t)
}

type NestedListConfig struct {
	Key0 float64      `json:"key0"`
	Key1 *NestedList1 `json:"key1"`
}
type NestedList1 struct {
	Key11 []*NestedList2 `json:"key11"`
}
type NestedList2 struct {
	KeyA string       `json:"keyA"`
	KeyB int          `json:"keyB"`
	KeyC []float64    `json:"keyC"`
	KeyD *NestedList3 `json:"keyD"`
}
type NestedList3 struct {
	KeyX string `json:"keyX"`
	KeyY string `json:"keyY"`
}

func TestNestedList(t *testing.T) {
	structValidation := &StructValidation{
		StructFieldValidations: []*StructFieldValidation{
			{
				// Key:               "key0",
				StructField:       "Key0",
				Float64Validation: &Float64Validation{},
			},
			{
				// Key:         "key1",
				StructField: "Key1",
				StructValidation: &StructValidation{
					StructFieldValidations: []*StructFieldValidation{
						{
							// Key:         "key11",
							StructField: "Key11",
							StructListValidation: &StructListValidation{
								StructValidation: &StructValidation{
									StructFieldValidations: []*StructFieldValidation{
										{
											// Key:              "keyA",
											StructField:      "KeyA",
											StringValidation: &StringValidation{},
										},
										{
											// Key:           "keyB",
											StructField:   "KeyB",
											IntValidation: &IntValidation{},
										},
										{
											// Key:                   "keyC",
											StructField:           "KeyC",
											Float64ListValidation: &Float64ListValidation{},
										},
										{
											// Key:         "keyD",
											StructField: "KeyD",
											StructValidation: &StructValidation{
												StructFieldValidations: []*StructFieldValidation{
													{
														// Key:              "keyX",
														StructField:      "KeyX",
														StringValidation: &StringValidation{},
													},
													{
														// Key:              "keyY",
														StructField:      "KeyY",
														StringValidation: &StringValidation{},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		Required:     true,
		ShortCircuit: true,
	}

	configData := MustReadYAMLStr(
		`
    key0: 1.1
    key1:
      key11:
        - keyA: A
          keyB: 0
          keyC:
            - 0.1
            - 0.2
          keyD:
            keyX: test1
            keyY: test2
        - keyA: X
          keyB: 1
          keyC:
            - 1.1
            - 1.2
            - 1.3
          keyD:
            keyX: test3
            keyY: test4
    `)

	expected := &NestedListConfig{
		Key0: 1.1,
		Key1: &NestedList1{
			Key11: []*NestedList2{
				{
					KeyA: "A",
					KeyB: 0,
					KeyC: []float64{float64(0.1), float64(0.2)},
					KeyD: &NestedList3{
						KeyX: "test1",
						KeyY: "test2",
					},
				},
				{
					KeyA: "X",
					KeyB: 1,
					KeyC: []float64{float64(1.1), float64(1.2), float64(1.3)},
					KeyD: &NestedList3{
						KeyX: "test3",
						KeyY: "test4",
					},
				},
			},
		},
	}

	testConfig(structValidation, configData, expected, t)
}

type Typed interface {
	GetType() string
}

type Typed1 struct {
	Key0 string `json:"key0"`
	Key1 string `json:"key1"`
}

type Typed1WithType struct {
	Type string `json:"type"`
	Key0 string `json:"key0"`
	Key1 string `json:"key1"`
}

func (t *Typed1) GetType() string {
	return "type1"
}

func (t *Typed1WithType) GetType() string {
	return "type1"
}

type Typed2 struct {
	KeyA int `json:"keyA"`
	KeyB int `json:"keyB"`
}

type Typed2WithType struct {
	Type string `json:"type"`
	KeyA int    `json:"keyA"`
	KeyB int    `json:"keyB"`
}

func (t *Typed2) GetType() string {
	return "type2"
}

func (t *Typed2WithType) GetType() string {
	return "type2"
}

type TypedConfig struct {
	Typed `json:"typed"`
}

var _interfaceStructValidation = &InterfaceStructValidation{
	TypeKey: "type",
	InterfaceStructTypes: map[string]*InterfaceStructType{
		"type1": {
			Type: (*Typed1)(nil),
			StructFieldValidations: []*StructFieldValidation{
				{
					// Key:              "key0",
					StructField:      "Key0",
					StringValidation: &StringValidation{},
				},
				{
					// Key:              "key1",
					StructField:      "Key1",
					StringValidation: &StringValidation{},
				},
			},
		},
		"type2": {
			Type: (*Typed2)(nil),
			StructFieldValidations: []*StructFieldValidation{
				{
					// Key:           "keyA",
					StructField:   "KeyA",
					IntValidation: &IntValidation{},
				},
				{
					// Key:           "keyB",
					StructField:   "KeyB",
					IntValidation: &IntValidation{},
				},
			},
		},
	},
}

var _interfaceStructValidationWithTypeKeyConfig = &InterfaceStructValidation{
	TypeKey:         "type",
	TypeStructField: "Type",
	InterfaceStructTypes: map[string]*InterfaceStructType{
		"type1": {
			Type: (*Typed1WithType)(nil),
			StructFieldValidations: []*StructFieldValidation{
				{
					// Key:              "key0",
					StructField:      "Key0",
					StringValidation: &StringValidation{},
				},
				{
					// Key:              "key1",
					StructField:      "Key1",
					StringValidation: &StringValidation{},
				},
			},
		},
		"type2": {
			Type: (*Typed2WithType)(nil),
			StructFieldValidations: []*StructFieldValidation{
				{
					// Key:           "keyA",
					StructField:   "KeyA",
					IntValidation: &IntValidation{},
				},
				{
					// Key:           "keyB",
					StructField:   "KeyB",
					IntValidation: &IntValidation{},
				},
			},
		},
	},
}

func TestInterface(t *testing.T) {
	structValidation := &StructValidation{
		StructFieldValidations: []*StructFieldValidation{
			{
				// Key:         "typed",
				StructField:               "Typed",
				InterfaceStructValidation: _interfaceStructValidation,
			},
		},
	}

	configDataType1 := MustReadYAMLStr(
		`
    typed:
      type: type1
      key0: testA
      key1: testB
    `)

	configDataType2 := MustReadYAMLStr(
		`
    typed:
      type: type2
      keyA: 0
      keyB: 1
    `)

	expectedType1 := &TypedConfig{
		Typed: &Typed1{
			Key0: "testA",
			Key1: "testB",
		},
	}

	expectedType2 := &TypedConfig{
		Typed: &Typed2{
			KeyA: 0,
			KeyB: 1,
		},
	}

	testConfig(structValidation, configDataType1, expectedType1, t)
	testConfig(structValidation, configDataType2, expectedType2, t)

	structValidation = &StructValidation{
		StructFieldValidations: []*StructFieldValidation{
			{
				// Key:         "typed",
				StructField:               "Typed",
				InterfaceStructValidation: _interfaceStructValidationWithTypeKeyConfig,
			},
		},
	}

	expectedTypeWithTypeKey1 := &TypedConfig{
		Typed: &Typed1WithType{
			Type: "type1",
			Key0: "testA",
			Key1: "testB",
		},
	}

	expectedTypeWithTypeKey2 := &TypedConfig{
		Typed: &Typed2WithType{
			Type: "type2",
			KeyA: 0,
			KeyB: 1,
		},
	}

	testConfig(structValidation, configDataType1, expectedTypeWithTypeKey1, t)
	testConfig(structValidation, configDataType2, expectedTypeWithTypeKey2, t)
}

type TypedListConfig struct {
	Typeds []Typed `json:"typeds"`
}

func TestInterfaceList(t *testing.T) {
	structValidation := &StructValidation{
		StructFieldValidations: []*StructFieldValidation{
			{
				// Key:         "typeds",
				StructField: "Typeds",
				InterfaceStructListValidation: &InterfaceStructListValidation{
					InterfaceStructValidation: _interfaceStructValidation,
				},
			},
		},
	}

	configData := MustReadYAMLStr(
		`
    typeds:
      - type: type1
        key0: testA
        key1: testB

      - type: type2
        keyA: 0
        keyB: 1

      - type: type1
        key0: test1
        key1: test2

      - type: type2
        keyA: 0
        keyB: -1
    `)

	expected := &TypedListConfig{
		Typeds: []Typed{
			&Typed1{
				Key0: "testA",
				Key1: "testB",
			},
			&Typed2{
				KeyA: 0,
				KeyB: 1,
			},
			&Typed1{
				Key0: "test1",
				Key1: "test2",
			},
			&Typed2{
				KeyA: 0,
				KeyB: -1,
			},
		},
	}

	testConfig(structValidation, configData, expected, t)

	structValidation = &StructValidation{
		StructFieldValidations: []*StructFieldValidation{
			{
				// Key:         "typeds",
				StructField: "Typeds",
				InterfaceStructListValidation: &InterfaceStructListValidation{
					InterfaceStructValidation: _interfaceStructValidationWithTypeKeyConfig,
				},
			},
		},
	}

	expected = &TypedListConfig{
		Typeds: []Typed{
			&Typed1WithType{
				Type: "type1",
				Key0: "testA",
				Key1: "testB",
			},
			&Typed2WithType{
				Type: "type2",
				KeyA: 0,
				KeyB: 1,
			},
			&Typed1WithType{
				Type: "type1",
				Key0: "test1",
				Key1: "test2",
			},
			&Typed2WithType{
				Type: "type2",
				KeyA: 0,
				KeyB: -1,
			},
		},
	}

	testConfig(structValidation, configData, expected, t)
}

type NullableConfig struct {
	Key1 *string     `json:"key1"`
	Key2 []string    `json:"key2"`
	Key3 interface{} `json:"key3"`
}

type NullableParentConfig struct {
	KeyA *NullableConfig `json:"key_a"`
}

func TestDefaultNull(t *testing.T) {
	configDataEmpty := MustReadYAMLStr(``)
	configDataNull := MustReadYAMLStr(`Null`)
	configDataEmptyMap := MustReadYAMLStr(`{}`)
	configDataNullValues := MustReadYAMLStr(
		`
     key1: null
     key2: null
     key3: null
     `)
	configDataParentNullValues := MustReadYAMLStr(
		`
     key_a:
       key1: null
       key2: null
       key3: null
     `)
	configDataParentNull := MustReadYAMLStr(
		`
     key_a: null
     `)

	structFieldValidations := []*StructFieldValidation{
		{
			StructField: "Key1",
			StringPtrValidation: &StringPtrValidation{
				AllowExplicitNull: true,
			},
		},
		{
			StructField: "Key2",
			StringListValidation: &StringListValidation{
				Default:           []string{"key2"},
				AllowExplicitNull: true,
			},
		},
		{
			StructField: "Key3",
			InterfaceValidation: &InterfaceValidation{
				Default:           "key3",
				AllowExplicitNull: true,
			},
		},
	}

	// AllowExplicitNull = true

	structValidation := &StructValidation{
		StructFieldValidations: structFieldValidations,
		AllowExplicitNull:      true,
	}

	var expected interface{}

	expected = &NullableConfig{}
	testConfig(structValidation, configDataEmpty, expected, t)
	testConfig(structValidation, configDataNull, expected, t)

	expected = &NullableConfig{
		Key1: nil,
		Key2: []string{"key2"},
		Key3: "key3",
	}
	testConfig(structValidation, configDataEmptyMap, expected, t)

	expected = &NullableConfig{
		Key1: nil,
		Key2: nil,
		Key3: nil,
	}
	testConfig(structValidation, configDataNullValues, expected, t)

	// AllowExplicitNull = false

	structValidation = &StructValidation{
		StructFieldValidations: structFieldValidations,
		AllowExplicitNull:      false,
	}

	expected = &NullableConfig{}
	testConfigError(structValidation, configDataEmpty, expected, t)
	testConfigError(structValidation, configDataNull, expected, t)

	expected = &NullableConfig{
		Key1: nil,
		Key2: []string{"key2"},
		Key3: "key3",
	}
	testConfig(structValidation, configDataEmptyMap, expected, t)

	expected = &NullableConfig{
		Key1: nil,
		Key2: nil,
		Key3: nil,
	}
	testConfig(structValidation, configDataNullValues, expected, t)

	// parent, AllowExplicitNull = true on both

	structValidation = &StructValidation{
		StructFieldValidations: []*StructFieldValidation{
			{
				StructField: "KeyA",
				StructValidation: &StructValidation{
					StructFieldValidations: structFieldValidations,
					AllowExplicitNull:      true,
				},
			},
		},
		AllowExplicitNull: true,
	}

	expected = &NullableParentConfig{}
	testConfig(structValidation, configDataEmpty, expected, t)
	testConfig(structValidation, configDataNull, expected, t)

	expected = &NullableParentConfig{
		KeyA: &NullableConfig{
			Key1: nil,
			Key2: []string{"key2"},
			Key3: "key3",
		},
	}
	testConfig(structValidation, configDataEmptyMap, expected, t)

	expected = &NullableParentConfig{}
	testConfig(structValidation, configDataParentNull, expected, t)

	expected = &NullableParentConfig{
		KeyA: &NullableConfig{
			Key1: nil,
			Key2: nil,
			Key3: nil,
		},
	}
	testConfig(structValidation, configDataParentNullValues, expected, t)

	// parent, AllowExplicitNull = false on both

	structValidation = &StructValidation{
		StructFieldValidations: []*StructFieldValidation{
			{
				StructField: "KeyA",
				StructValidation: &StructValidation{
					StructFieldValidations: structFieldValidations,
					AllowExplicitNull:      false,
				},
			},
		},
		AllowExplicitNull: false,
	}

	expected = &NullableParentConfig{}
	testConfigError(structValidation, configDataEmpty, expected, t)
	testConfigError(structValidation, configDataNull, expected, t)

	expected = &NullableParentConfig{
		KeyA: &NullableConfig{
			Key1: nil,
			Key2: []string{"key2"},
			Key3: "key3",
		},
	}
	testConfig(structValidation, configDataEmptyMap, expected, t)

	testConfigError(structValidation, configDataParentNull, expected, t)

	expected = &NullableParentConfig{
		KeyA: &NullableConfig{
			Key1: nil,
			Key2: nil,
			Key3: nil,
		},
	}
	testConfig(structValidation, configDataParentNullValues, expected, t)

	// parent, AllowExplicitNull = true on child, DefaultNil = true

	structValidation = &StructValidation{
		StructFieldValidations: []*StructFieldValidation{
			{
				StructField: "KeyA",
				StructValidation: &StructValidation{
					StructFieldValidations: structFieldValidations,
					AllowExplicitNull:      true,
					DefaultNil:             true,
				},
			},
		},
		AllowExplicitNull: false,
	}

	expected = &NullableParentConfig{}
	testConfigError(structValidation, configDataEmpty, expected, t)
	testConfigError(structValidation, configDataNull, expected, t)

	expected = &NullableParentConfig{}
	testConfig(structValidation, configDataEmptyMap, expected, t)

	expected = &NullableParentConfig{}
	testConfig(structValidation, configDataParentNull, expected, t)

	expected = &NullableParentConfig{
		KeyA: &NullableConfig{
			Key1: nil,
			Key2: nil,
			Key3: nil,
		},
	}
	testConfig(structValidation, configDataParentNullValues, expected, t)

	// parent, AllowExplicitNull = false on both, DefaultNil = true

	structValidation = &StructValidation{
		StructFieldValidations: []*StructFieldValidation{
			{
				StructField: "KeyA",
				StructValidation: &StructValidation{
					StructFieldValidations: structFieldValidations,
					AllowExplicitNull:      false,
					DefaultNil:             true,
				},
			},
		},
		AllowExplicitNull: false,
	}

	expected = &NullableParentConfig{}
	testConfigError(structValidation, configDataEmpty, expected, t)
	testConfigError(structValidation, configDataNull, expected, t)

	expected = &NullableParentConfig{}
	testConfig(structValidation, configDataEmptyMap, expected, t)

	expected = &NullableParentConfig{}
	testConfigError(structValidation, configDataParentNull, expected, t)

	expected = &NullableParentConfig{
		KeyA: &NullableConfig{
			Key1: nil,
			Key2: nil,
			Key3: nil,
		},
	}
	testConfig(structValidation, configDataParentNullValues, expected, t)
}

type DefaultConfig struct {
	Key1 bool   `json:"key1"`
	Key2 string `json:"key2"`
	Key3 string `json:"key3"`
}

func TestDefaultField(t *testing.T) {
	structValidation := &StructValidation{
		StructFieldValidations: []*StructFieldValidation{
			{
				StructField:    "Key1",
				BoolValidation: &BoolValidation{},
			},
			{
				StructField:      "Key2",
				StringValidation: &StringValidation{},
			},
			{
				StructField:      "Key3",
				DefaultField:     "Key2",
				StringValidation: &StringValidation{},
			},
		},
	}

	configData := MustReadYAMLStr(
		`
    key1: true
    key2: "key2"
    key3: "key3"
    `)
	expected := &DefaultConfig{
		Key1: true,
		Key2: "key2",
		Key3: "key3",
	}
	testConfig(structValidation, configData, expected, t)

	configData = MustReadYAMLStr(
		`
    key1: true
    key2: "key2"
    `)
	expected = &DefaultConfig{
		Key1: true,
		Key2: "key2",
		Key3: "key2",
	}
	testConfig(structValidation, configData, expected, t)

	structValidation = &StructValidation{
		StructFieldValidations: []*StructFieldValidation{
			{
				StructField:    "Key1",
				BoolValidation: &BoolValidation{},
			},
			{
				StructField:      "Key2",
				StringValidation: &StringValidation{},
			},
			{
				StructField:            "Key3",
				DefaultDependentFields: []string{"Key2"},
				DefaultDependentFieldsFunc: func(vals []interface{}) interface{} {
					return vals[0].(string) + ".py"
				},
				StringValidation: &StringValidation{},
			},
		},
	}

	configData = MustReadYAMLStr(
		`
    key1: true
    key2: "key2"
    `)
	expected = &DefaultConfig{
		Key1: true,
		Key2: "key2",
		Key3: "key2.py",
	}
	testConfig(structValidation, configData, expected, t)

	structValidation = &StructValidation{
		StructFieldValidations: []*StructFieldValidation{
			{
				StructField:    "Key1",
				BoolValidation: &BoolValidation{},
			},
			{
				StructField:      "Key2",
				StringValidation: &StringValidation{},
			},
			{
				StructField:            "Key3",
				DefaultDependentFields: []string{"Key1"},
				DefaultDependentFieldsFunc: func(vals []interface{}) interface{} {
					if vals[0].(bool) {
						return "It was true"
					}
					return "It was false"
				},
				StringValidation: &StringValidation{},
			},
		},
	}

	configData = MustReadYAMLStr(
		`
    key1: true
    key2: "key2"
    `)
	expected = &DefaultConfig{
		Key1: true,
		Key2: "key2",
		Key3: "It was true",
	}
	testConfig(structValidation, configData, expected, t)

	structValidation = &StructValidation{
		StructFieldValidations: []*StructFieldValidation{
			{
				StructField:      "Key2",
				StringValidation: &StringValidation{},
			},
			{
				StructField:      "Key3",
				StringValidation: &StringValidation{},
			},
			{
				StructField:            "Key1",
				DefaultDependentFields: []string{"Key2"},
				DefaultDependentFieldsFunc: func(vals []interface{}) interface{} {
					if vals[0].(string) == "key2" {
						return true
					}
					return false
				},
				BoolValidation: &BoolValidation{},
			},
		},
	}

	configData = MustReadYAMLStr(
		`
    key2: "key2"
    key3: "key3"
    `)
	expected = &DefaultConfig{
		Key1: true,
		Key2: "key2",
		Key3: "key3",
	}
	testConfig(structValidation, configData, expected, t)

	configData = MustReadYAMLStr(
		`
    key2: "test"
    key3: "key3"
    `)
	expected = &DefaultConfig{
		Key1: false,
		Key2: "test",
		Key3: "key3",
	}
	testConfig(structValidation, configData, expected, t)
}

func testConfig(structValidation *StructValidation, configData interface{}, expected interface{}, t *testing.T) {
	config := reflect.New(reflect.TypeOf(expected).Elem()).Interface()

	errs := Struct(config, configData, structValidation)

	if errs != nil {
		for _, err := range errs {
			fmt.Println("ERROR: " + errors.Message(err))
		}
	}
	require.Empty(t, errs)

	require.Equal(t, expected, config)
}

func testConfigError(structValidation *StructValidation, configData interface{}, expectedTypeInstance interface{}, t *testing.T) {
	config := reflect.New(reflect.TypeOf(expectedTypeInstance).Elem()).Interface()
	errs := Struct(config, configData, structValidation)
	require.NotEmpty(t, errs)
}
