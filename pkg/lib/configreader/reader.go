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
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/cortexlabs/cortex/pkg/lib/cast"
	"github.com/cortexlabs/cortex/pkg/lib/debug"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/lib/maps"
	"github.com/cortexlabs/cortex/pkg/lib/prompt"
	"github.com/cortexlabs/cortex/pkg/lib/slices"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/yaml"
)

type StructFieldValidation struct {
	Key                        string                          // Required, defaults to json key or "StructField"
	StructField                string                          // Required
	DefaultField               string                          // Optional. Will set the default to the runtime value of this field
	DefaultDependentFields     []string                        // Optional. Will be passed in to DefaultDependentFieldsFunc. Dependent fields must be listed first in the `[]*cr.StructFieldValidation`.
	DefaultDependentFieldsFunc func([]interface{}) interface{} // Optional. Will be called with DefaultDependentFields

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
	AllowExplicitNull      bool
	TreatNullAsEmpty       bool // If explicit null or if it's top level and the file is empty, treat as empty map
	DefaultNil             bool // If this struct is nested and its key is not defined, set it to nil instead of defaults or erroring (e.g. if any subfields are required)
	CantBeSpecifiedErrStr  *string
	ShortCircuit           bool
	AllowExtraFields       bool
}

type StructListValidation struct {
	StructValidation      *StructValidation
	Required              bool
	AllowExplicitNull     bool
	TreatNullAsEmpty      bool // If explicit null or if it's top level and the file is empty, treat as empty map
	MinLength             int
	MaxLength             int
	InvalidLengths        []int
	CantBeSpecifiedErrStr *string
	ShortCircuit          bool
}

type InterfaceStructValidation struct {
	TypeKey                    string                               // required
	TypeStructField            string                               // optional (will set this field if present)
	InterfaceStructTypes       map[string]*InterfaceStructType      // specify this or ParsedInterfaceStructTypes
	ParsedInterfaceStructTypes map[interface{}]*InterfaceStructType // must specify Parser if using this
	Parser                     func(string) (interface{}, error)
	Required                   bool
	AllowExplicitNull          bool
	TreatNullAsEmpty           bool // If explicit null or if it's top level and the file is empty, treat as empty map
	CantBeSpecifiedErrStr      *string
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
	AllowExplicitNull         bool
	TreatNullAsEmpty          bool // If explicit null or if it's top level and the file is empty, treat as empty map
	CantBeSpecifiedErrStr     *string
	ShortCircuit              bool
}

func Struct(dest interface{}, inter interface{}, v *StructValidation) []error {
	allowedFields := []string{}
	allErrs := []error{}
	var ok bool

	if inter == nil {
		if v.TreatNullAsEmpty {
			inter = make(map[interface{}]interface{}, 0)
		} else {
			if !v.AllowExplicitNull {
				return []error{ErrorCannotBeEmptyOrNull(v.Required)}
			}
			return nil
		}
	}

	interMap, ok := cast.InterfaceToStrInterfaceMap(inter)
	if !ok {
		return []error{ErrorInvalidPrimitiveType(inter, PrimTypeMap)}
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
			if ok && validation.CantBeSpecifiedErrStr != nil {
				err = errors.Wrap(ErrorFieldCantBeSpecified(*validation.CantBeSpecifiedErrStr), key)
			} else if !ok && validation.Required {
				err = errors.Wrap(ErrorMustBeDefined(), key)
			} else if !ok && validation.DefaultNil {
				val = nil
			} else {
				if !ok {
					interMapVal = make(map[string]interface{}) // Here validation.DefaultNil == false, so create an empty map to hold the nested default values
				}
				val = reflect.New(nestedType.Elem()).Interface()
				errs = Struct(val, interMapVal, &validation)
				if interMapVal == nil {
					val = nil // If the object was nil, set val to nil rather than a pointer to the initialized zero value
				}
				errs = errors.WrapAll(errs, key)
			}

		} else if structFieldValidation.StructListValidation != nil {
			validation := *structFieldValidation.StructListValidation
			updateValidation(&validation, dest, structFieldValidation)
			nestedType := reflect.ValueOf(dest).Elem().FieldByName(structFieldValidation.StructField).Type()
			interMapVal, ok := ReadInterfaceMapValue(key, interMap)
			if ok && validation.CantBeSpecifiedErrStr != nil {
				err = errors.Wrap(ErrorFieldCantBeSpecified(*validation.CantBeSpecifiedErrStr), key)
			} else if !ok && validation.Required {
				err = errors.Wrap(ErrorMustBeDefined(), key)
			} else {
				val = reflect.Indirect(reflect.New(nestedType)).Interface()
				val, errs = StructList(val, interMapVal, &validation)
				errs = errors.WrapAll(errs, key)
			}

		} else if structFieldValidation.InterfaceStructValidation != nil {
			validation := *structFieldValidation.InterfaceStructValidation
			updateValidation(&validation, dest, structFieldValidation)
			interMapVal, ok := ReadInterfaceMapValue(key, interMap)
			if ok && validation.CantBeSpecifiedErrStr != nil {
				err = errors.Wrap(ErrorFieldCantBeSpecified(*validation.CantBeSpecifiedErrStr), key)
			} else if !ok && validation.Required {
				err = errors.Wrap(ErrorMustBeDefined(), key)
			} else {
				val, errs = InterfaceStruct(interMapVal, &validation)
				errs = errors.WrapAll(errs, key)
			}

		} else if structFieldValidation.InterfaceStructListValidation != nil {
			validation := *structFieldValidation.InterfaceStructListValidation
			updateValidation(&validation, dest, structFieldValidation)
			nestedType := reflect.ValueOf(dest).Elem().FieldByName(structFieldValidation.StructField).Type()
			interMapVal, ok := ReadInterfaceMapValue(key, interMap)
			if ok && validation.CantBeSpecifiedErrStr != nil {
				err = errors.Wrap(ErrorFieldCantBeSpecified(*validation.CantBeSpecifiedErrStr), key)
			} else if !ok && validation.Required {
				err = errors.Wrap(ErrorMustBeDefined(), key)
			} else {
				val = reflect.Indirect(reflect.New(nestedType)).Interface()
				val, errs = InterfaceStructList(val, interMapVal, &validation)
				errs = errors.WrapAll(errs, key)
			}

		} else {
			exit.Panic(ErrorUnsupportedFieldValidation())
		}

		allErrs, _ = errors.AddError(allErrs, err)
		allErrs, _ = errors.AddErrors(allErrs, errs)
		if errors.HasError(allErrs) {
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
	if errors.HasError(allErrs) {
		return allErrs
	}
	return nil
}

func StructList(dest interface{}, inter interface{}, v *StructListValidation) (interface{}, []error) {
	if inter == nil {
		if v.TreatNullAsEmpty {
			inter = make([]interface{}, 0)
		} else {
			if !v.AllowExplicitNull {
				return nil, []error{ErrorCannotBeEmptyOrNull(v.Required)}
			}
			return nil, nil
		}
	}

	interSlice, ok := cast.InterfaceToInterfaceSlice(inter)
	if !ok {
		return nil, []error{ErrorInvalidPrimitiveType(inter, PrimTypeList)}
	}

	if v.MinLength != 0 {
		if len(interSlice) < v.MinLength {
			return nil, []error{ErrorTooFewElements(v.MinLength)}
		}
	}
	if v.MaxLength != 0 {
		if len(interSlice) > v.MaxLength {
			return nil, []error{ErrorTooManyElements(v.MaxLength)}
		}
	}
	for _, invalidLength := range v.InvalidLengths {
		if len(interSlice) == invalidLength {
			return nil, []error{ErrorWrongNumberOfElements(v.InvalidLengths)}
		}
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
		if interItem == nil {
			val = nil // If the object was nil, set val to nil rather than a pointer to the initialized zero value
		}
		dest = appendVal(dest, val)
	}

	return dest, errs
}

func InterfaceStruct(inter interface{}, v *InterfaceStructValidation) (interface{}, []error) {
	if inter == nil {
		if v.TreatNullAsEmpty {
			inter = make(map[interface{}]interface{}, 0)
		} else {
			if !v.AllowExplicitNull {
				return nil, []error{ErrorCannotBeEmptyOrNull(v.Required)}
			}
			return nil, nil
		}
	}

	interMap, ok := cast.InterfaceToStrInterfaceMap(inter)
	if !ok {
		return nil, []error{ErrorInvalidPrimitiveType(inter, PrimTypeMap)}
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
			return nil, []error{errors.Wrap(ErrorInvalidInterface(typeStr, validTypeObjs[0], validTypeObjs[1:]...), v.TypeKey)}
		}
	}

	val := reflect.New(reflect.TypeOf(structType.Type).Elem()).Interface()
	structValidation := &StructValidation{
		StructFieldValidations: append(structType.StructFieldValidations, typeFieldValidation),
		Required:               v.Required,
		AllowExplicitNull:      v.AllowExplicitNull,
		ShortCircuit:           v.ShortCircuit,
		AllowExtraFields:       v.AllowExtraFields,
	}
	errs := Struct(val, inter, structValidation)
	return val, errs
}

func InterfaceStructList(dest interface{}, inter interface{}, v *InterfaceStructListValidation) (interface{}, []error) {
	if inter == nil {
		if v.TreatNullAsEmpty {
			inter = make([]interface{}, 0)
		} else {
			if !v.AllowExplicitNull {
				return nil, []error{ErrorCannotBeEmptyOrNull(v.Required)}
			}
			return nil, nil
		}
	}

	interSlice, ok := cast.InterfaceToInterfaceSlice(inter)
	if !ok {
		return nil, []error{ErrorInvalidPrimitiveType(inter, PrimTypeList)}
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
		setField(runtimeVal, validation, "Default")
	} else if structFieldValidation.DefaultDependentFieldsFunc != nil {
		runtimeVals := make([]interface{}, len(structFieldValidation.DefaultDependentFields))
		for i, fieldName := range structFieldValidation.DefaultDependentFields {
			runtimeVals[i] = reflect.ValueOf(dest).Elem().FieldByName(fieldName).Interface()
		}
		val := structFieldValidation.DefaultDependentFieldsFunc(runtimeVals)
		setField(val, validation, "Default")
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

type PromptItemValidation struct {
	StructField string          // Required
	PromptOpts  *prompt.Options // Required

	// Provide one of the following:
	StringValidation     *StringValidation
	StringPtrValidation  *StringPtrValidation
	BoolValidation       *BoolValidation
	BoolPtrValidation    *BoolPtrValidation
	IntValidation        *IntValidation
	IntPtrValidation     *IntPtrValidation
	Int32Validation      *Int32Validation
	Int32PtrValidation   *Int32PtrValidation
	Int64Validation      *Int64Validation
	Int64PtrValidation   *Int64PtrValidation
	Float32Validation    *Float32Validation
	Float32PtrValidation *Float32PtrValidation
	Float64Validation    *Float64Validation
	Float64PtrValidation *Float64PtrValidation

	// Additional parsing step for StringValidation or StringPtrValidation
	Parser func(string) (interface{}, error)
}

type PromptValidation struct {
	PromptItemValidations  []*PromptItemValidation
	SkipNonEmptyFields     bool // skips fields that are not zero-valued
	SkipNonNilFields       bool // skips pointer fields that are not nil
	PrintNewLineIfPrompted bool // prints an extra new line at the end if any questions were asked
}

func ReadPrompt(dest interface{}, promptValidation *PromptValidation) error {
	var val interface{}
	var err error
	shouldPrintTrailingNewLine := false

	// Validate any skipped fields first, so that any errors are returned before prompting
	if promptValidation.SkipNonEmptyFields {
		for _, promptItemValidation := range promptValidation.PromptItemValidations {
			v := reflect.ValueOf(dest).Elem().FieldByName(promptItemValidation.StructField)
			if !v.IsZero() {
				if promptItemValidation.StringValidation != nil && promptItemValidation.Parser == nil {
					if _, err := ValidateStringProvided(v.Interface().(string), promptItemValidation.StringValidation); err != nil {
						return errors.Wrap(err, inferPromptFieldName(reflect.TypeOf(dest), promptItemValidation.StructField))
					}
				} else if promptItemValidation.StringPtrValidation != nil && promptItemValidation.Parser == nil {
					if _, err := ValidateStringPtrProvided(v.Interface().(*string), promptItemValidation.StringPtrValidation); err != nil {
						return errors.Wrap(err, inferPromptFieldName(reflect.TypeOf(dest), promptItemValidation.StructField))
					}
				} else if promptItemValidation.BoolValidation != nil {
					if _, err := ValidateBoolProvided(v.Interface().(bool), promptItemValidation.BoolValidation); err != nil {
						return errors.Wrap(err, inferPromptFieldName(reflect.TypeOf(dest), promptItemValidation.StructField))
					}
				} else if promptItemValidation.BoolPtrValidation != nil {
					if _, err := ValidateBoolPtrProvided(v.Interface().(*bool), promptItemValidation.BoolPtrValidation); err != nil {
						return errors.Wrap(err, inferPromptFieldName(reflect.TypeOf(dest), promptItemValidation.StructField))
					}
				} else if promptItemValidation.IntValidation != nil {
					if _, err := ValidateIntProvided(v.Interface().(int), promptItemValidation.IntValidation); err != nil {
						return errors.Wrap(err, inferPromptFieldName(reflect.TypeOf(dest), promptItemValidation.StructField))
					}
				} else if promptItemValidation.IntPtrValidation != nil {
					if _, err := ValidateIntPtrProvided(v.Interface().(*int), promptItemValidation.IntPtrValidation); err != nil {
						return errors.Wrap(err, inferPromptFieldName(reflect.TypeOf(dest), promptItemValidation.StructField))
					}
				} else if promptItemValidation.Int32Validation != nil {
					if _, err := ValidateInt32Provided(v.Interface().(int32), promptItemValidation.Int32Validation); err != nil {
						return errors.Wrap(err, inferPromptFieldName(reflect.TypeOf(dest), promptItemValidation.StructField))
					}
				} else if promptItemValidation.Int32PtrValidation != nil {
					if _, err := ValidateInt32PtrProvided(v.Interface().(*int32), promptItemValidation.Int32PtrValidation); err != nil {
						return errors.Wrap(err, inferPromptFieldName(reflect.TypeOf(dest), promptItemValidation.StructField))
					}
				} else if promptItemValidation.Int64Validation != nil {
					if _, err := ValidateInt64Provided(v.Interface().(int64), promptItemValidation.Int64Validation); err != nil {
						return errors.Wrap(err, inferPromptFieldName(reflect.TypeOf(dest), promptItemValidation.StructField))
					}
				} else if promptItemValidation.Int64PtrValidation != nil {
					if _, err := ValidateInt64PtrProvided(v.Interface().(*int64), promptItemValidation.Int64PtrValidation); err != nil {
						return errors.Wrap(err, inferPromptFieldName(reflect.TypeOf(dest), promptItemValidation.StructField))
					}
				} else if promptItemValidation.Float32Validation != nil {
					if _, err := ValidateFloat32Provided(v.Interface().(float32), promptItemValidation.Float32Validation); err != nil {
						return errors.Wrap(err, inferPromptFieldName(reflect.TypeOf(dest), promptItemValidation.StructField))
					}
				} else if promptItemValidation.Float32PtrValidation != nil {
					if _, err := ValidateFloat32PtrProvided(v.Interface().(*float32), promptItemValidation.Float32PtrValidation); err != nil {
						return errors.Wrap(err, inferPromptFieldName(reflect.TypeOf(dest), promptItemValidation.StructField))
					}
				} else if promptItemValidation.Float64Validation != nil {
					if _, err := ValidateFloat64Provided(v.Interface().(float64), promptItemValidation.Float64Validation); err != nil {
						return errors.Wrap(err, inferPromptFieldName(reflect.TypeOf(dest), promptItemValidation.StructField))
					}
				} else if promptItemValidation.Float64PtrValidation != nil {
					if _, err := ValidateFloat64PtrProvided(v.Interface().(*float64), promptItemValidation.Float64PtrValidation); err != nil {
						return errors.Wrap(err, inferPromptFieldName(reflect.TypeOf(dest), promptItemValidation.StructField))
					}
				}
			}
		}
	}

	for _, promptItemValidation := range promptValidation.PromptItemValidations {
		if promptValidation.SkipNonEmptyFields {
			v := reflect.ValueOf(dest).Elem().FieldByName(promptItemValidation.StructField)
			if !v.IsZero() {
				continue
			}
		} else if promptValidation.SkipNonNilFields {
			v := reflect.ValueOf(dest).Elem().FieldByName(promptItemValidation.StructField)
			if v.Kind() == reflect.Ptr && !v.IsNil() {
				continue
			}
		}

		if promptValidation.PrintNewLineIfPrompted {
			shouldPrintTrailingNewLine = true
		}

		for {
			if promptItemValidation.StringValidation != nil {
				val, err = StringFromPrompt(promptItemValidation.PromptOpts, promptItemValidation.StringValidation)
				if err == nil && promptItemValidation.Parser != nil {
					val, err = promptItemValidation.Parser(val.(string))
				}
			} else if promptItemValidation.StringPtrValidation != nil {
				val, err = StringPtrFromPrompt(promptItemValidation.PromptOpts, promptItemValidation.StringPtrValidation)
				if err == nil && promptItemValidation.Parser != nil {
					if val.(*string) == nil {
						val = nil
					} else {
						val, err = promptItemValidation.Parser(*val.(*string))
						if err == nil && val != nil {
							valValue := reflect.ValueOf(val)
							valPtrValue := reflect.New(valValue.Type())
							valPtrValue.Elem().Set(valValue)
							val = valPtrValue.Interface()
						} else {
							val = nil
							// err is already set
						}
					}
				}
			} else if promptItemValidation.BoolValidation != nil {
				val, err = BoolFromPrompt(promptItemValidation.PromptOpts, promptItemValidation.BoolValidation)
			} else if promptItemValidation.BoolPtrValidation != nil {
				val, err = BoolPtrFromPrompt(promptItemValidation.PromptOpts, promptItemValidation.BoolPtrValidation)
			} else if promptItemValidation.IntValidation != nil {
				val, err = IntFromPrompt(promptItemValidation.PromptOpts, promptItemValidation.IntValidation)
			} else if promptItemValidation.IntPtrValidation != nil {
				val, err = IntPtrFromPrompt(promptItemValidation.PromptOpts, promptItemValidation.IntPtrValidation)
			} else if promptItemValidation.Int32Validation != nil {
				val, err = Int32FromPrompt(promptItemValidation.PromptOpts, promptItemValidation.Int32Validation)
			} else if promptItemValidation.Int32PtrValidation != nil {
				val, err = Int32PtrFromPrompt(promptItemValidation.PromptOpts, promptItemValidation.Int32PtrValidation)
			} else if promptItemValidation.Int64Validation != nil {
				val, err = Int64FromPrompt(promptItemValidation.PromptOpts, promptItemValidation.Int64Validation)
			} else if promptItemValidation.Int64PtrValidation != nil {
				val, err = Int64PtrFromPrompt(promptItemValidation.PromptOpts, promptItemValidation.Int64PtrValidation)
			} else if promptItemValidation.Float32Validation != nil {
				val, err = Float32FromPrompt(promptItemValidation.PromptOpts, promptItemValidation.Float32Validation)
			} else if promptItemValidation.Float32PtrValidation != nil {
				val, err = Float32PtrFromPrompt(promptItemValidation.PromptOpts, promptItemValidation.Float32PtrValidation)
			} else if promptItemValidation.Float64Validation != nil {
				val, err = Float64FromPrompt(promptItemValidation.PromptOpts, promptItemValidation.Float64Validation)
			} else if promptItemValidation.Float64PtrValidation != nil {
				val, err = Float64PtrFromPrompt(promptItemValidation.PromptOpts, promptItemValidation.Float64PtrValidation)
			} else {
				exit.Panic(ErrorUnsupportedFieldValidation())
			}

			if err == nil {
				break
			}

			if promptItemValidation.PromptOpts.SkipTrailingNewline {
				fmt.Printf("error: %s\n", errors.Message(err))
			} else {
				fmt.Printf("error: %s\n\n", errors.Message(err))
			}
		}

		if val == nil {
			err = setFieldNil(dest, promptItemValidation.StructField)
		} else {
			err = setField(val, dest, promptItemValidation.StructField)
		}

		if err != nil {
			return err
		}
	}

	if shouldPrintTrailingNewLine {
		fmt.Println()
	}

	return nil
}

// Reads a string map into a struct
func StructFromStringMap(dest interface{}, strMap map[string]string, v *StructValidation) []error {
	allowedFields := []string{}
	allErrs := []error{}
	var ok bool

	if strMap == nil {
		if v.TreatNullAsEmpty {
			strMap = make(map[string]string, 0)
		} else {
			if !v.AllowExplicitNull {
				return []error{ErrorCannotBeEmptyOrNull(v.Required)}
			}
			return nil
		}
	}

	for _, structFieldValidation := range v.StructFieldValidations {
		key := inferKey(reflect.TypeOf(dest), structFieldValidation.StructField, structFieldValidation.Key)
		allowedFields = append(allowedFields, key)

		if structFieldValidation.Nil == true {
			continue
		}

		strMapVal, keyExists := strMap[key]

		var err error
		var errs []error
		var val interface{}

		if structFieldValidation.StringValidation != nil {
			validation := *structFieldValidation.StringValidation
			updateValidation(&validation, dest, structFieldValidation)
			if keyExists {
				val, err = StringFromStr(strMapVal, &validation)
			} else {
				val, err = ValidateStringMissing(&validation)
			}
			if err == nil && structFieldValidation.Parser != nil {
				val, err = structFieldValidation.Parser(val.(string))
			}
		} else if structFieldValidation.StringPtrValidation != nil {
			validation := *structFieldValidation.StringPtrValidation
			updateValidation(&validation, dest, structFieldValidation)
			if keyExists {
				val, err = StringPtrFromStr(strMapVal, &validation)
			} else {
				val, err = ValidateStringPtrMissing(&validation)
			}
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
					}
				}
			}
		} else if structFieldValidation.BoolValidation != nil {
			validation := *structFieldValidation.BoolValidation
			updateValidation(&validation, dest, structFieldValidation)
			if keyExists {
				val, err = BoolFromStr(strMapVal, &validation)
			} else {
				val, err = ValidateBoolMissing(&validation)
			}
		} else if structFieldValidation.BoolPtrValidation != nil {
			validation := *structFieldValidation.BoolPtrValidation
			updateValidation(&validation, dest, structFieldValidation)
			if keyExists {
				val, err = BoolPtrFromStr(strMapVal, &validation)
			} else {
				val, err = ValidateBoolPtrMissing(&validation)
			}
		} else if structFieldValidation.IntValidation != nil {
			validation := *structFieldValidation.IntValidation
			updateValidation(&validation, dest, structFieldValidation)
			if keyExists {
				val, err = IntFromStr(strMapVal, &validation)
			} else {
				val, err = ValidateIntMissing(&validation)
			}
		} else if structFieldValidation.IntPtrValidation != nil {
			validation := *structFieldValidation.IntPtrValidation
			updateValidation(&validation, dest, structFieldValidation)
			if keyExists {
				val, err = IntPtrFromStr(strMapVal, &validation)
			} else {
				val, err = ValidateIntPtrMissing(&validation)
			}
		} else if structFieldValidation.Int32Validation != nil {
			validation := *structFieldValidation.Int32Validation
			updateValidation(&validation, dest, structFieldValidation)
			if keyExists {
				val, err = Int32FromStr(strMapVal, &validation)
			} else {
				val, err = ValidateInt32Missing(&validation)
			}
		} else if structFieldValidation.Int32PtrValidation != nil {
			validation := *structFieldValidation.Int32PtrValidation
			updateValidation(&validation, dest, structFieldValidation)
			if keyExists {
				val, err = Int32PtrFromStr(strMapVal, &validation)
			} else {
				val, err = ValidateInt32PtrMissing(&validation)
			}
		} else if structFieldValidation.Int64Validation != nil {
			validation := *structFieldValidation.Int64Validation
			updateValidation(&validation, dest, structFieldValidation)
			if keyExists {
				val, err = Int64FromStr(strMapVal, &validation)
			} else {
				val, err = ValidateInt64Missing(&validation)
			}
		} else if structFieldValidation.Int64PtrValidation != nil {
			validation := *structFieldValidation.Int64PtrValidation
			updateValidation(&validation, dest, structFieldValidation)
			if keyExists {
				val, err = Int64PtrFromStr(strMapVal, &validation)
			} else {
				val, err = ValidateInt64PtrMissing(&validation)
			}
		} else if structFieldValidation.Float32Validation != nil {
			validation := *structFieldValidation.Float32Validation
			updateValidation(&validation, dest, structFieldValidation)
			if keyExists {
				val, err = Float32FromStr(strMapVal, &validation)
			} else {
				val, err = ValidateFloat32Missing(&validation)
			}
		} else if structFieldValidation.Float32PtrValidation != nil {
			validation := *structFieldValidation.Float32PtrValidation
			updateValidation(&validation, dest, structFieldValidation)
			if keyExists {
				val, err = Float32PtrFromStr(strMapVal, &validation)
			} else {
				val, err = ValidateFloat32PtrMissing(&validation)
			}
		} else if structFieldValidation.Float64Validation != nil {
			validation := *structFieldValidation.Float64Validation
			updateValidation(&validation, dest, structFieldValidation)
			if keyExists {
				val, err = Float64FromStr(strMapVal, &validation)
			} else {
				val, err = ValidateFloat64Missing(&validation)
			}
		} else if structFieldValidation.Float64PtrValidation != nil {
			validation := *structFieldValidation.Float64PtrValidation
			updateValidation(&validation, dest, structFieldValidation)
			if keyExists {
				val, err = Float64PtrFromStr(strMapVal, &validation)
			} else {
				val, err = ValidateFloat64PtrMissing(&validation)
			}
		} else {
			exit.Panic(ErrorUnsupportedFieldValidation())
		}

		err = errors.Wrap(err, key)
		errs = errors.WrapAll(errs, key)

		allErrs, _ = errors.AddError(allErrs, err)
		allErrs, _ = errors.AddErrors(allErrs, errs)
		if errors.HasError(allErrs) {
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
		extraFields := slices.SubtractStrSlice(maps.StrMapKeysString(strMap), allowedFields)
		for _, extraField := range extraFields {
			allErrs = append(allErrs, ErrorUnsupportedKey(extraField))
		}
	}
	if errors.HasError(allErrs) {
		return allErrs
	}
	return nil
}

// Reads a directory of files into a struct, where each file name is the key and the contents is the value
func StructFromFiles(dest interface{}, dirPath string, v *StructValidation) []error {
	strMap := map[string]string{}

	fileNames, err := files.ListDir(dirPath, true)
	if err != nil {
		return []error{err}
	}

	for _, fileName := range fileNames {
		fileBytes, err := files.ReadFileBytes(filepath.Join(dirPath, fileName))
		if err != nil {
			return []error{err}
		}
		strMap[fileName] = strings.TrimSpace(string(fileBytes))
	}

	return StructFromStringMap(dest, strMap, v)
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

func ParseYAMLFile(dest interface{}, validation *StructValidation, filePath string) []error {
	fileInterface, err := ReadYAMLFile(filePath)
	if err != nil {
		return []error{err}
	}

	errs := Struct(dest, fileInterface, validation)
	if errors.HasError(errs) {
		return errors.WrapAll(errs, filePath)
	}

	return nil
}

func ReadYAMLFile(filePath string) (interface{}, error) {
	fileBytes, err := files.ReadFileBytes(filePath)
	if err != nil {
		return nil, err
	}

	fileInterface, err := ReadYAMLBytes(fileBytes)
	if err != nil {
		return nil, errors.Wrap(err, filePath)
	}

	return fileInterface, nil
}

func ReadYAMLFileStrMap(filePath string) (map[string]interface{}, error) {
	parsed, err := ReadYAMLFile(filePath)
	if err != nil {
		return nil, err
	}
	casted, ok := cast.InterfaceToStrInterfaceMap(parsed)
	if !ok {
		return nil, ErrorInvalidPrimitiveType(parsed, PrimTypeMap)
	}
	return casted, nil
}

func ReadYAMLBytes(yamlBytes []byte) (interface{}, error) {
	if len(yamlBytes) == 0 {
		return nil, nil
	}
	var parsed interface{}
	err := yaml.Unmarshal(yamlBytes, &parsed)
	if err != nil {
		return nil, ErrorInvalidYAML(err)
	}
	return parsed, nil
}

func ReadJSONBytes(jsonBytes []byte) (interface{}, error) {
	if len(jsonBytes) == 0 {
		return nil, nil
	}
	var parsed interface{}
	err := json.DecodeWithNumber(jsonBytes, &parsed)
	if err != nil {
		return nil, err
	}
	return parsed, nil
}

func MustReadYAMLStr(yamlStr string) interface{} {
	parsed, err := ReadYAMLBytes([]byte(yamlStr))
	if err != nil {
		exit.Panic(err)
	}
	return parsed
}

func MustReadYAMLStrMap(yamlStr string) map[string]interface{} {
	parsed, err := ReadYAMLBytes([]byte(yamlStr))
	if err != nil {
		exit.Panic(err)
	}
	casted, ok := cast.InterfaceToStrInterfaceMap(parsed)
	if !ok {
		exit.Panic(ErrorInvalidPrimitiveType(parsed, PrimTypeMap))
	}
	return casted
}

func MustReadJSONStr(jsonStr string) interface{} {
	parsed, err := ReadJSONBytes([]byte(jsonStr))
	if err != nil {
		exit.Panic(err)
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
		debug.Ppg(val)
		debug.Ppg(destStruct)
		return errors.Wrap(ErrorCannotSetStructField(), fieldName)
	}

	if val == nil {
		// Check for nil-able types
		if v.Kind() == reflect.Chan || v.Kind() == reflect.Func || v.Kind() == reflect.Interface || v.Kind() == reflect.Map || v.Kind() == reflect.Ptr || v.Kind() == reflect.Slice {
			v.Set(reflect.Zero(v.Type()))
			return nil
		}
		debug.Ppg(val)
		debug.Ppg(destStruct)
		return errors.Wrap(ErrorCannotSetStructField(), fieldName)
	}

	if !reflect.ValueOf(val).Type().AssignableTo(v.Type()) {
		debug.Ppg(val)
		debug.Ppg(destStruct)
		return errors.Wrap(ErrorCannotSetStructField(), fieldName)
	}

	v.Set(reflect.ValueOf(val))
	return nil
}

// destStruct must be a pointer to a struct
func setFirstField(val interface{}, destStruct interface{}) error {
	v := reflect.ValueOf(destStruct).Elem().FieldByIndex([]int{0})
	if !v.IsValid() || !v.CanSet() {
		debug.Ppg(val)
		debug.Ppg(destStruct)
		return errors.Wrap(ErrorCannotSetStructField(), "first field")
	}
	v.Set(reflect.ValueOf(val))
	return nil
}

// destStruct must be a pointer to a struct
func setFieldNil(destStruct interface{}, fieldName string) error {
	v := reflect.ValueOf(destStruct).Elem().FieldByName(fieldName)
	if !v.IsValid() || !v.CanSet() {
		debug.Ppg(destStruct)
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

func inferPromptFieldName(structType reflect.Type, typeStructField string) string {
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
