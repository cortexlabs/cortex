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

package configreader

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	input "github.com/tcnksm/go-input"
	yaml "gopkg.in/yaml.v2"

	s "github.com/cortexlabs/cortex/pkg/api/strings"
	"github.com/cortexlabs/cortex/pkg/lib/cast"
	"github.com/cortexlabs/cortex/pkg/lib/debug"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	libjson "github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/lib/maps"
	"github.com/cortexlabs/cortex/pkg/lib/slices"
)

type StructFieldValidation struct {
	Key              string                        // Required, defaults to json key or "StructField"
	StructField      string                        // Required
	DefaultField     string                        // Optional. Will set the default to the runtime value of this field
	DefaultFieldFunc func(interface{}) interface{} // Optional. Will call the func with the value of DefaultField

	// Provide one of the following:
	StringValidation              *StringValidation
	StringPtrValidation           *StringPtrValidation
	StringListValidation          *StringListValidation
	BoolValidation                *BoolValidation
	BoolPtrValidation             *BoolPtrValidation
	BoolListValidation            *BoolListValidation
	IntValidation                 *IntValidation
	IntPtrValidation              *IntPtrValidation
	IntListValidation             *IntListValidation
	Int32Validation               *Int32Validation
	Int32PtrValidation            *Int32PtrValidation
	Int32ListValidation           *Int32ListValidation
	Int64Validation               *Int64Validation
	Int64PtrValidation            *Int64PtrValidation
	Int64ListValidation           *Int64ListValidation
	Float32Validation             *Float32Validation
	Float32PtrValidation          *Float32PtrValidation
	Float32ListValidation         *Float32ListValidation
	Float64Validation             *Float64Validation
	Float64PtrValidation          *Float64PtrValidation
	Float64ListValidation         *Float64ListValidation
	StringMapValidation           *StringMapValidation
	InterfaceMapValidation        *InterfaceMapValidation
	InterfaceMapListValidation    *InterfaceMapListValidation
	InterfaceValidation           *InterfaceValidation
	StructValidation              *StructValidation
	StructListValidation          *StructListValidation
	InterfaceStructValidation     *InterfaceStructValidation
	InterfaceStructListValidation *InterfaceStructListValidation
	Nil                           bool

	// Additional parsing step for StringValidation or StringPtrValidation
	Parser func(string) (interface{}, error)
}

type StructValidation struct {
	StructFieldValidations []*StructFieldValidation
	Required               bool
	AllowNull              bool
	DefualtNil             bool // If this struct is nested and it's key is not defined, set it to nil instead of defaults or erroring (e.g. if any subfields are required)
	ShortCircuit           bool
	AllowExtraFields       bool
}

type StructListValidation struct {
	StructValidation *StructValidation
	Required         bool
	AllowNull        bool
	ShortCircuit     bool
}

type InterfaceStructValidation struct {
	TypeKey                    string                               // required
	TypeStructField            string                               // optional (will set this field if present)
	InterfaceStructTypes       map[string]*InterfaceStructType      // specify this or ParsedInterfaceStructTypes
	ParsedInterfaceStructTypes map[interface{}]*InterfaceStructType // must specify Parser if using this
	Parser                     func(string) (interface{}, error)
	Required                   bool
	AllowNull                  bool
	ShortCircuit               bool
	AllowExtraFields           bool
}

type InterfaceStructType struct {
	Type                   interface{} // e.g. (*MyType)(nil)
	StructFieldValidations []*StructFieldValidation
}

type InterfaceStructListValidation struct {
	InterfaceStructValidation *InterfaceStructValidation
	Required                  bool
	AllowNull                 bool
	ShortCircuit              bool
}

func Struct(dest interface{}, inter interface{}, v *StructValidation) []error {
	allowedFields := []string{}
	allErrs := []error{}
	var ok bool

	if inter == nil {
		if !v.AllowNull {
			return []error{ErrorCannotBeNull()}
		}
	}

	interMap, ok := cast.InterfaceToStrInterfaceMap(inter)
	if !ok {
		return []error{ErrorInvalidPrimitiveType(inter, s.PrimTypeMap)}
	}

	for _, structFieldValidation := range v.StructFieldValidations {
		key := inferKey(reflect.TypeOf(dest), structFieldValidation.StructField, structFieldValidation.Key)
		allowedFields = append(allowedFields, key)

		if structFieldValidation.Nil == true {
			continue
		}

		var err error
		var errs []error
		var val interface{}

		if structFieldValidation.StringValidation != nil {
			validation := *structFieldValidation.StringValidation
			updateValidation(&validation, dest, structFieldValidation)
			val, err = StringFromInterfaceMap(key, interMap, &validation)
			if err == nil && structFieldValidation.Parser != nil {
				val, err = structFieldValidation.Parser(val.(string))
				err = errors.Wrap(err, key)
			}
		} else if structFieldValidation.StringPtrValidation != nil {
			validation := *structFieldValidation.StringPtrValidation
			updateValidation(&validation, dest, structFieldValidation)
			val, err = StringPtrFromInterfaceMap(key, interMap, &validation)
			if err == nil && structFieldValidation.Parser != nil {
				if val.(*string) == nil {
					val = nil
				} else {
					val, err = structFieldValidation.Parser(*val.(*string))
					if err == nil && val != nil {
						valValue := reflect.ValueOf(val)
						valPtrValue := reflect.New(valValue.Type())
						valPtrValue.Elem().Set(valValue)
						val = valPtrValue.Interface()
					} else {
						val = nil
						err = errors.Wrap(err, key)
					}
				}
			}
		} else if structFieldValidation.StringListValidation != nil {
			validation := *structFieldValidation.StringListValidation
			updateValidation(&validation, dest, structFieldValidation)
			val, err = StringListFromInterfaceMap(key, interMap, &validation)
		} else if structFieldValidation.BoolValidation != nil {
			validation := *structFieldValidation.BoolValidation
			updateValidation(&validation, dest, structFieldValidation)
			val, err = BoolFromInterfaceMap(key, interMap, &validation)
		} else if structFieldValidation.BoolPtrValidation != nil {
			validation := *structFieldValidation.BoolPtrValidation
			updateValidation(&validation, dest, structFieldValidation)
			val, err = BoolPtrFromInterfaceMap(key, interMap, &validation)
		} else if structFieldValidation.BoolListValidation != nil {
			validation := *structFieldValidation.BoolListValidation
			updateValidation(&validation, dest, structFieldValidation)
			val, err = BoolListFromInterfaceMap(key, interMap, &validation)
		} else if structFieldValidation.IntValidation != nil {
			validation := *structFieldValidation.IntValidation
			updateValidation(&validation, dest, structFieldValidation)
			val, err = IntFromInterfaceMap(key, interMap, &validation)
		} else if structFieldValidation.IntPtrValidation != nil {
			validation := *structFieldValidation.IntPtrValidation
			updateValidation(&validation, dest, structFieldValidation)
			val, err = IntPtrFromInterfaceMap(key, interMap, &validation)
		} else if structFieldValidation.IntListValidation != nil {
			validation := *structFieldValidation.IntListValidation
			updateValidation(&validation, dest, structFieldValidation)
			val, err = IntListFromInterfaceMap(key, interMap, &validation)
		} else if structFieldValidation.Int32Validation != nil {
			validation := *structFieldValidation.Int32Validation
			updateValidation(&validation, dest, structFieldValidation)
			val, err = Int32FromInterfaceMap(key, interMap, &validation)
		} else if structFieldValidation.Int32PtrValidation != nil {
			validation := *structFieldValidation.Int32PtrValidation
			updateValidation(&validation, dest, structFieldValidation)
			val, err = Int32PtrFromInterfaceMap(key, interMap, &validation)
		} else if structFieldValidation.Int32ListValidation != nil {
			validation := *structFieldValidation.Int32ListValidation
			updateValidation(&validation, dest, structFieldValidation)
			val, err = Int32ListFromInterfaceMap(key, interMap, &validation)
		} else if structFieldValidation.Int64Validation != nil {
			validation := *structFieldValidation.Int64Validation
			updateValidation(&validation, dest, structFieldValidation)
			val, err = Int64FromInterfaceMap(key, interMap, &validation)
		} else if structFieldValidation.Int64PtrValidation != nil {
			validation := *structFieldValidation.Int64PtrValidation
			updateValidation(&validation, dest, structFieldValidation)
			val, err = Int64PtrFromInterfaceMap(key, interMap, &validation)
		} else if structFieldValidation.Int64ListValidation != nil {
			validation := *structFieldValidation.Int64ListValidation
			updateValidation(&validation, dest, structFieldValidation)
			val, err = Int64ListFromInterfaceMap(key, interMap, &validation)
		} else if structFieldValidation.Float32Validation != nil {
			validation := *structFieldValidation.Float32Validation
			updateValidation(&validation, dest, structFieldValidation)
			val, err = Float32FromInterfaceMap(key, interMap, &validation)
		} else if structFieldValidation.Float32PtrValidation != nil {
			validation := *structFieldValidation.Float32PtrValidation
			updateValidation(&validation, dest, structFieldValidation)
			val, err = Float32PtrFromInterfaceMap(key, interMap, &validation)
		} else if structFieldValidation.Float64Validation != nil {
			validation := *structFieldValidation.Float64Validation
			updateValidation(&validation, dest, structFieldValidation)
			val, err = Float64FromInterfaceMap(key, interMap, &validation)
		} else if structFieldValidation.Float64PtrValidation != nil {
			validation := *structFieldValidation.Float64PtrValidation
			updateValidation(&validation, dest, structFieldValidation)
			val, err = Float64PtrFromInterfaceMap(key, interMap, &validation)
		} else if structFieldValidation.Float32ListValidation != nil {
			validation := *structFieldValidation.Float32ListValidation
			updateValidation(&validation, dest, structFieldValidation)
			val, err = Float32ListFromInterfaceMap(key, interMap, &validation)
		} else if structFieldValidation.Float64ListValidation != nil {
			validation := *structFieldValidation.Float64ListValidation
			updateValidation(&validation, dest, structFieldValidation)
			val, err = Float64ListFromInterfaceMap(key, interMap, &validation)
		} else if structFieldValidation.StringMapValidation != nil {
			validation := *structFieldValidation.StringMapValidation
			updateValidation(&validation, dest, structFieldValidation)
			val, err = StringMapFromInterfaceMap(key, interMap, &validation)
		} else if structFieldValidation.InterfaceMapValidation != nil {
			validation := *structFieldValidation.InterfaceMapValidation
			updateValidation(&validation, dest, structFieldValidation)
			val, err = InterfaceMapFromInterfaceMap(key, interMap, &validation)
		} else if structFieldValidation.InterfaceMapListValidation != nil {
			validation := *structFieldValidation.InterfaceMapListValidation
			updateValidation(&validation, dest, structFieldValidation)
			val, err = InterfaceMapListFromInterfaceMap(key, interMap, &validation)
		} else if structFieldValidation.InterfaceValidation != nil {
			validation := *structFieldValidation.InterfaceValidation
			updateValidation(&validation, dest, structFieldValidation)
			val, err = InterfaceFromInterfaceMap(key, interMap, &validation)
		} else if structFieldValidation.StructValidation != nil {
			validation := *structFieldValidation.StructValidation
			updateValidation(&validation, dest, structFieldValidation)
			nestedType := reflect.ValueOf(dest).Elem().FieldByName(structFieldValidation.StructField).Type()
			interMapVal, ok := ReadInterfaceMapValue(key, interMap)
			if !ok && validation.Required {
				err = errors.Wrap(ErrorMustBeDefined(), key)
			} else if !ok && validation.DefualtNil {
				val = nil
			} else {
				if !ok {
					interMapVal = make(map[string]interface{}) // Distinguish between null and not defined
				}
				val = reflect.New(nestedType.Elem()).Interface()
				errs = Struct(val, interMapVal, &validation)
				errs = errors.WrapMultiple(errs, key)
			}

		} else if structFieldValidation.StructListValidation != nil {
			validation := *structFieldValidation.StructListValidation
			updateValidation(&validation, dest, structFieldValidation)
			nestedType := reflect.ValueOf(dest).Elem().FieldByName(structFieldValidation.StructField).Type()
			interMapVal, ok := ReadInterfaceMapValue(key, interMap)
			if !ok && validation.Required {
				err = errors.Wrap(ErrorMustBeDefined(), key)
			} else {
				val = reflect.Indirect(reflect.New(nestedType)).Interface()
				val, errs = StructList(val, interMapVal, &validation)
				errs = errors.WrapMultiple(errs, key)
			}

		} else if structFieldValidation.InterfaceStructValidation != nil {
			validation := *structFieldValidation.InterfaceStructValidation
			updateValidation(&validation, dest, structFieldValidation)
			interMapVal, ok := ReadInterfaceMapValue(key, interMap)
			if !ok && validation.Required {
				err = errors.Wrap(ErrorMustBeDefined(), key)
			} else {
				val, errs = InterfaceStruct(interMapVal, &validation)
				errs = errors.WrapMultiple(errs, key)
			}

		} else if structFieldValidation.InterfaceStructListValidation != nil {
			validation := *structFieldValidation.InterfaceStructListValidation
			updateValidation(&validation, dest, structFieldValidation)
			nestedType := reflect.ValueOf(dest).Elem().FieldByName(structFieldValidation.StructField).Type()
			interMapVal, ok := ReadInterfaceMapValue(key, interMap)
			if !ok && validation.Required {
				err = errors.Wrap(ErrorMustBeDefined(), key)
			} else {
				val = reflect.Indirect(reflect.New(nestedType)).Interface()
				val, errs = InterfaceStructList(val, interMapVal, &validation)
				errs = errors.WrapMultiple(errs, key)
			}

		} else {
			errors.Panic("Undefined or unsupported validation type for ReadInterfaceMap")
		}

		allErrs, _ = errors.AddError(allErrs, err)
		allErrs, _ = errors.AddErrors(allErrs, errs)
		if errors.HasErrors(allErrs) {
			if v.ShortCircuit {
				return allErrs
			}
			continue
		}

		if val == nil {
			err = setFieldNil(dest, structFieldValidation.StructField)
		} else {
			err = setField(val, dest, structFieldValidation.StructField)
		}
		if allErrs, ok = errors.AddError(allErrs, err, key); ok {
			if v.ShortCircuit {
				return allErrs
			}
		}
	}

	if !v.AllowExtraFields {
		extraFields := slices.SubtractStrSlice(maps.InterfaceMapKeys(interMap), allowedFields)
		for _, extraField := range extraFields {
			allErrs = append(allErrs, ErrorUnsupportedKey(extraField))
		}
	}
	if errors.HasErrors(allErrs) {
		return allErrs
	}
	return nil
}

func StructList(dest interface{}, inter interface{}, v *StructListValidation) (interface{}, []error) {
	if inter == nil {
		if !v.AllowNull {
			return nil, []error{ErrorCannotBeNull()}
		}
		return nil, nil
	}

	interSlice, ok := cast.InterfaceToInterfaceSlice(inter)
	if !ok {
		return nil, []error{ErrorInvalidPrimitiveType(inter, s.PrimTypeList)}
	}

	errs := []error{}
	for i, interItem := range interSlice {
		val := reflect.New(reflect.ValueOf(dest).Type().Elem().Elem()).Interface()
		subErrs := Struct(val, interItem, v.StructValidation)
		var ok bool
		if errs, ok = errors.AddErrors(errs, subErrs, s.Index(i)); ok {
			if v.ShortCircuit {
				return nil, errs
			}
			continue
		}
		dest = appendVal(dest, val)
	}

	return dest, errs
}

func InterfaceStruct(inter interface{}, v *InterfaceStructValidation) (interface{}, []error) {
	if inter == nil {
		if !v.AllowNull {
			return nil, []error{ErrorCannotBeNull()}
		}
		return nil, nil
	}

	interMap, ok := cast.InterfaceToStrInterfaceMap(inter)
	if !ok {
		return nil, []error{ErrorInvalidPrimitiveType(inter, s.PrimTypeMap)}
	}

	var validTypeStrs []string
	if v.InterfaceStructTypes != nil {
		for typeStr := range v.InterfaceStructTypes {
			validTypeStrs = append(validTypeStrs, typeStr)
		}
	}

	typeStrValidation := &StringValidation{
		Required:      true,
		AllowedValues: validTypeStrs,
	}

	typeStr, err := StringFromInterfaceMap(v.TypeKey, interMap, typeStrValidation)
	if err != nil {
		return nil, []error{err}
	}
	var typeObj interface{}
	if v.Parser != nil {
		typeObj, err = v.Parser(typeStr)
		if err != nil {
			return nil, []error{errors.Wrap(err, v.TypeKey)}
		}
	}

	var typeFieldValidation *StructFieldValidation
	if v.TypeStructField == "" {
		typeFieldValidation = &StructFieldValidation{
			Key: v.TypeKey,
			Nil: true,
		}
	} else {
		typeFieldValidation = &StructFieldValidation{
			Key:              v.TypeKey,
			StructField:      v.TypeStructField,
			StringValidation: typeStrValidation,
			Parser:           v.Parser,
		}
	}

	var structType *InterfaceStructType
	if v.InterfaceStructTypes != nil {
		structType = v.InterfaceStructTypes[typeStr]
	} else {
		structType = v.ParsedInterfaceStructTypes[typeObj]
		if structType == nil {
			// This error case may or may not be handled by v.Parser()
			var validTypeObjs []interface{}
			for typeObj := range v.ParsedInterfaceStructTypes {
				validTypeObjs = append(validTypeObjs, typeObj)
			}
			return nil, []error{errors.Wrap(ErrorInvalidInterface(typeStr, validTypeObjs...), v.TypeKey)}
		}
	}

	val := reflect.New(reflect.TypeOf(structType.Type).Elem()).Interface()
	structValidation := &StructValidation{
		StructFieldValidations: append(structType.StructFieldValidations, typeFieldValidation),
		Required:               v.Required,
		AllowNull:              v.AllowNull,
		ShortCircuit:           v.ShortCircuit,
		AllowExtraFields:       v.AllowExtraFields,
	}
	errs := Struct(val, inter, structValidation)
	return val, errs
}

func InterfaceStructList(dest interface{}, inter interface{}, v *InterfaceStructListValidation) (interface{}, []error) {
	if inter == nil {
		if !v.AllowNull {
			return nil, []error{ErrorCannotBeNull()}
		}
		return nil, nil
	}

	interSlice, ok := cast.InterfaceToInterfaceSlice(inter)
	if !ok {
		return nil, []error{ErrorInvalidPrimitiveType(inter, s.PrimTypeList)}
	}

	errs := []error{}
	for i, interItem := range interSlice {
		val, subErrs := InterfaceStruct(interItem, v.InterfaceStructValidation)
		var ok bool
		if errs, ok = errors.AddErrors(errs, subErrs, s.Index(i)); ok {
			if v.ShortCircuit {
				return nil, errs
			}
			continue
		}
		dest = appendVal(dest, val)
	}

	return dest, errs
}

func updateValidation(validation interface{}, dest interface{}, structFieldValidation *StructFieldValidation) {
	if structFieldValidation.DefaultField != "" {
		runtimeVal := reflect.ValueOf(dest).Elem().FieldByName(structFieldValidation.DefaultField).Interface()
		if structFieldValidation.DefaultFieldFunc != nil {
			runtimeVal = structFieldValidation.DefaultFieldFunc(runtimeVal)
		}
		setField(runtimeVal, validation, "Default")
	}
}

func ReadInterfaceMapValue(name string, interMap map[string]interface{}) (interface{}, bool) {
	if interMap == nil {
		return nil, false
	}

	val, ok := interMap[name]
	if !ok {
		return nil, false
	}
	return val, true
}

//
// Prompt
//

var ui *input.UI = &input.UI{
	Writer: os.Stdout,
	Reader: os.Stdin,
}

type PromptItemValidation struct {
	StructField string         // Required
	PromptOpts  *PromptOptions // Required

	// Provide one of the following:
	StringValidation  *StringValidation
	BoolValidation    *BoolValidation
	IntValidation     *IntValidation
	Int32Validation   *Int32Validation
	Int64Validation   *Int64Validation
	Float32Validation *Float32Validation
	Float64Validation *Float64Validation
}

type PromptValidation struct {
	PromptItemValidations []*PromptItemValidation
}

func ReadPrompt(dest interface{}, promptValidation *PromptValidation) error {
	var val interface{}
	var err error

	for _, promptItemValidation := range promptValidation.PromptItemValidations {
		for {
			if promptItemValidation.StringValidation != nil {
				val, err = StringFromPrompt(promptItemValidation.PromptOpts, promptItemValidation.StringValidation)
			} else if promptItemValidation.BoolValidation != nil {
				val, err = BoolFromPrompt(promptItemValidation.PromptOpts, promptItemValidation.BoolValidation)
			} else if promptItemValidation.IntValidation != nil {
				val, err = IntFromPrompt(promptItemValidation.PromptOpts, promptItemValidation.IntValidation)
			} else if promptItemValidation.Int32Validation != nil {
				val, err = Int32FromPrompt(promptItemValidation.PromptOpts, promptItemValidation.Int32Validation)
			} else if promptItemValidation.Int64Validation != nil {
				val, err = Int64FromPrompt(promptItemValidation.PromptOpts, promptItemValidation.Int64Validation)
			} else if promptItemValidation.Float32Validation != nil {
				val, err = Float32FromPrompt(promptItemValidation.PromptOpts, promptItemValidation.Float32Validation)
			} else if promptItemValidation.Float64Validation != nil {
				val, err = Float64FromPrompt(promptItemValidation.PromptOpts, promptItemValidation.Float64Validation)
			} else {
				errors.Panic("Undefined or unsupported validation type for ReadPrompt")
			}

			if err == nil {
				break
			}
			fmt.Println(err.Error())
		}

		err = setField(val, dest, promptItemValidation.StructField)
		if err != nil {
			return err
		}
	}

	return nil
}

type PromptOptions struct {
	Prompt        string
	MaskDefault   bool
	HideTyping    bool
	MaskTyping    bool
	TypingMaskVal string
	defaultStr    string
}

func prompt(opts *PromptOptions) string {
	prompt := opts.Prompt

	if opts.defaultStr != "" {
		defualtStr := opts.defaultStr
		if opts.MaskDefault {
			defualtStr = s.MaskString(defualtStr, 4)
		}
		prompt = fmt.Sprintf("%s [%s]", opts.Prompt, defualtStr)
	}

	val, err := ui.Ask(prompt, &input.Options{
		Default:     opts.defaultStr,
		Hide:        opts.HideTyping,
		Mask:        opts.MaskTyping,
		MaskVal:     opts.TypingMaskVal,
		Required:    false,
		HideDefault: true,
		HideOrder:   true,
		Loop:        false,
	})

	if err != nil {
		errors.Panic(err)
	}

	return val
}

//
// Environment variable
//

func ReadEnvVar(envVarName string) *string {
	envVar, envVarIsSet := os.LookupEnv(envVarName)
	if envVarIsSet {
		return &envVar
	}
	return nil
}

//
// JSON and YAML Config
//

func ReadYAMLBytes(yamlBytes []byte) (interface{}, error) {
	if len(yamlBytes) == 0 {
		return nil, nil
	}
	var parsed interface{}
	err := yaml.Unmarshal(yamlBytes, &parsed)
	if err != nil {
		return nil, ErrorUnmarshalYAML(err)
	}
	return parsed, nil
}

func ReadJSONBytes(jsonBytes []byte) (interface{}, error) {
	if len(jsonBytes) == 0 {
		return nil, nil
	}
	var parsed interface{}
	err := libjson.DecodeWithNumber(jsonBytes, &parsed)
	if err != nil {
		return nil, err
	}
	return parsed, nil
}

func MustReadYAMLStr(yamlStr string) interface{} {
	parsed, err := ReadYAMLBytes([]byte(yamlStr))
	if err != nil {
		errors.Panic(err)
	}
	return parsed
}

func MustReadYAMLStrMap(yamlStr string) map[string]interface{} {
	parsed, err := ReadYAMLBytes([]byte(yamlStr))
	if err != nil {
		errors.Panic(err)
	}
	casted, ok := cast.InterfaceToStrInterfaceMap(parsed)
	if !ok {
		errors.Panic(ErrorInvalidPrimitiveType(parsed, s.PrimTypeMap))
	}
	return casted
}

func MustReadJSONStr(jsonStr string) interface{} {
	parsed, err := ReadJSONBytes([]byte(jsonStr))
	if err != nil {
		errors.Panic(err)
	}
	return parsed
}

//
// Helpers
//

func appendVal(slice interface{}, val interface{}) interface{} {
	return reflect.Append(reflect.ValueOf(slice), reflect.ValueOf(val)).Interface()
}

// destStruct must be a pointer to a struct
func setField(val interface{}, destStruct interface{}, fieldName string) error {
	v := reflect.ValueOf(destStruct).Elem().FieldByName(fieldName)
	if !v.IsValid() || !v.CanSet() {
		debug.Pp(val)
		debug.Pp(destStruct)
		return errors.Wrap(ErrorCannotSetStructField(), fieldName)
	}
	if !reflect.ValueOf(val).Type().AssignableTo(v.Type()) {
		debug.Pp(val)
		debug.Pp(destStruct)
		return errors.Wrap(ErrorCannotSetStructField(), fieldName)
	}
	v.Set(reflect.ValueOf(val))
	return nil
}

// destStruct must be a pointer to a struct
func setFirstField(val interface{}, destStruct interface{}) error {
	v := reflect.ValueOf(destStruct).Elem().FieldByIndex([]int{0})
	if !v.IsValid() || !v.CanSet() {
		debug.Pp(val)
		debug.Pp(destStruct)
		return errors.Wrap(ErrorCannotSetStructField(), "first field")
	}
	v.Set(reflect.ValueOf(val))
	return nil
}

// destStruct must be a pointer to a struct
func setFieldNil(destStruct interface{}, fieldName string) error {
	v := reflect.ValueOf(destStruct).Elem().FieldByName(fieldName)
	if !v.IsValid() || !v.CanSet() {
		debug.Pp(destStruct)
		return errors.Wrap(ErrorCannotSetStructField(), fieldName)
	}
	v.Set(reflect.Zero(v.Type()))
	return nil
}

// destStruct must be a pointer to a struct
func setFieldIfExists(val interface{}, destStruct interface{}, fieldName string) bool {
	if structHasKey(destStruct, fieldName) {
		err := setField(val, destStruct, fieldName)
		return err == nil
	}
	return false
}

// structVal must be a pointer to a struct
func structHasKey(val interface{}, fieldName string) bool {
	v := reflect.ValueOf(val).Elem().FieldByName(fieldName)
	if v.IsValid() && v.CanSet() {
		return true
	}
	return false
}

func inferKey(structType reflect.Type, typeStructField string, typeKey string) string {
	if typeKey != "" {
		return typeKey
	}
	field, _ := structType.Elem().FieldByName(typeStructField)
	tag, ok := getTagFieldName(field)
	if ok {
		return tag
	}
	return typeStructField
}

func getTagFieldName(field reflect.StructField) (string, bool) {
	tag, ok := field.Tag.Lookup("json")
	if ok {
		return strings.Split(tag, ",")[0], true
	}
	tag, ok = field.Tag.Lookup("yaml")
	if ok {
		return strings.Split(tag, ",")[0], true
	}
	return "", false
}
