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

package strings

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/cortexlabs/yaml"
)

func Bool(val bool) string {
	return strconv.FormatBool(val)
}

func Float32(val float32) string {
	str := strconv.FormatFloat(float64(val), 'f', -1, 32)
	if !strings.Contains(str, ".") {
		str = str + ".0"
	}
	return str
}

func Float64(val float64) string {
	str := strconv.FormatFloat(val, 'f', -1, 64)
	if !strings.Contains(str, ".") {
		str = str + ".0"
	}
	return str
}

func Int(val int) string {
	return strconv.Itoa(val)
}

func Int64(val int64) string {
	return strconv.FormatInt(val, 10)
}

func Int32(val int32) string {
	return strconv.FormatInt(int64(val), 10)
}

func Int16(val int16) string {
	return strconv.FormatInt(int64(val), 10)
}

func Int8(val int8) string {
	return strconv.FormatInt(int64(val), 10)
}

func Uint(val uint) string {
	return strconv.FormatUint(uint64(val), 10)
}

func Uint8(val uint8) string {
	return strconv.FormatUint(uint64(val), 10)
}

func Uint16(val uint16) string {
	return strconv.FormatUint(uint64(val), 10)
}

func Uint32(val uint32) string {
	return strconv.FormatUint(uint64(val), 10)
}

func Uint64(val uint64) string {
	return strconv.FormatUint(val, 10)
}

func Complex64(val complex64) string {
	return fmt.Sprint(val)
}

func Complex128(val complex128) string {
	return fmt.Sprint(val)
}

func Uintptr(val uintptr) string {
	return fmt.Sprint(val)
}

func Round(val float64, decimalPlaces int, padToDecimalPlaces int) string {
	rounded := math.Round(val*math.Pow10(decimalPlaces)) / math.Pow10(decimalPlaces)
	str := strconv.FormatFloat(rounded, 'f', -1, 64)
	if padToDecimalPlaces == 0 {
		return str
	}
	split := strings.Split(str, ".")
	intVal := split[0]
	decVal := ""
	if len(split) > 1 {
		decVal = split[1]
	}
	if len(decVal) >= padToDecimalPlaces {
		return str
	}
	numZeros := padToDecimalPlaces - len(decVal)
	return intVal + "." + decVal + strings.Repeat("0", numZeros)
}

// copied from https://yourbasic.org/golang/formatting-byte-size-to-human-readable-format/
func IntToBase2Byte(size int) string {
	return Int64ToBase2Byte(int64(size))
}

func Int64ToBase2Byte(size int64) string {
	const unit int64 = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB",
		float64(size)/float64(div), "KMGTPE"[exp])

}

func DollarsAndCents(val float64) string {
	return "$" + Round(val, 2, 2)
}

func DollarsAndTenthsOfCents(val float64) string {
	return "$" + Round(val, 3, 2)
}

func DollarsMaxPrecision(val float64) string {
	return "$" + Round(val, 100, 2)
}

// This is similar to json.Marshal, but handles non-string keys (which we support). It should be valid YAML since we use it in templates
func strIndent(val interface{}, indent string, currentIndent string, newlineChar string, quoteStr string) string {
	if val == nil {
		return "<null>"
	}

	value := reflect.ValueOf(val)
	valueType := value.Type()

	if value.Kind() == reflect.Invalid {
		return "<invalid>"
	}

	if value.Kind() == reflect.Chan || value.Kind() == reflect.Func || value.Kind() == reflect.Interface || value.Kind() == reflect.Map || value.Kind() == reflect.Ptr || value.Kind() == reflect.Slice {
		if value.IsNil() {
			return "<null>"
		}
	}

	stringSep := "," + newlineChar
	if len(newlineChar) == 0 {
		stringSep += " "
	}

	// Use a String() method if one exists
	funcVal := value.MethodByName("String")
	if funcVal.IsValid() {
		t := funcVal.Type()
		if t.NumIn() == 0 && t.NumOut() == 1 && t.Out(0).Kind() == reflect.String {
			return strIndent(funcVal.Call(nil)[0].Interface().(string), indent, currentIndent, newlineChar, quoteStr)
		}
	}
	if _, ok := reflect.PtrTo(valueType).MethodByName("String"); ok {
		ptrValue := reflect.New(valueType)
		ptrValue.Elem().Set(value)
		funcVal := ptrValue.MethodByName("String")
		if funcVal.IsValid() {
			t := funcVal.Type()
			if t.NumIn() == 0 && t.NumOut() == 1 && t.Out(0).Kind() == reflect.String {
				return strIndent(funcVal.Call(nil)[0].Interface().(string), indent, currentIndent, newlineChar, quoteStr)
			}
		}
	}

	switch value.Kind() {

	case reflect.Bool:
		var t bool
		return Bool(value.Convert(reflect.TypeOf(t)).Interface().(bool))
	case reflect.Float32:
		var t float32
		return Float32(value.Convert(reflect.TypeOf(t)).Interface().(float32))
	case reflect.Float64:
		var t float64
		return Float64(value.Convert(reflect.TypeOf(t)).Interface().(float64))
	case reflect.Int:
		var t int
		return Int(value.Convert(reflect.TypeOf(t)).Interface().(int))
	case reflect.Int8:
		var t int8
		return Int8(value.Convert(reflect.TypeOf(t)).Interface().(int8))
	case reflect.Int16:
		var t int16
		return Int16(value.Convert(reflect.TypeOf(t)).Interface().(int16))
	case reflect.Int32:
		var t int32
		return Int32(value.Convert(reflect.TypeOf(t)).Interface().(int32))
	case reflect.Int64:
		var t int64
		return Int64(value.Convert(reflect.TypeOf(t)).Interface().(int64))
	case reflect.Uint:
		var t uint
		return Uint(value.Convert(reflect.TypeOf(t)).Interface().(uint))
	case reflect.Uint8:
		var t uint8
		return Uint8(value.Convert(reflect.TypeOf(t)).Interface().(uint8))
	case reflect.Uint16:
		var t uint16
		return Uint16(value.Convert(reflect.TypeOf(t)).Interface().(uint16))
	case reflect.Uint32:
		var t uint32
		return Uint32(value.Convert(reflect.TypeOf(t)).Interface().(uint32))
	case reflect.Uint64:
		var t uint64
		return Uint64(value.Convert(reflect.TypeOf(t)).Interface().(uint64))
	case reflect.Complex64:
		var t complex64
		return Complex64(value.Convert(reflect.TypeOf(t)).Interface().(complex64))
	case reflect.Complex128:
		var t complex128
		return Complex128(value.Convert(reflect.TypeOf(t)).Interface().(complex128))
	case reflect.Uintptr:
		var t uintptr
		return Uintptr(value.Convert(reflect.TypeOf(t)).Interface().(uintptr))

	case reflect.String:
		var t string
		casted := value.Convert(reflect.TypeOf(t)).Interface().(string)

		var ok bool
		casted, ok = yaml.UnescapeAtSymbolOk(casted)
		if ok {
			return casted
		}

		switch val.(type) {
		case json.Number:
			return casted
		default:
			return quoteStr + casted + quoteStr
		}

	case reflect.Slice:
		fallthrough
	case reflect.Array:
		if value.Len() == 0 {
			return "[]"
		}
		strs := make([]string, value.Len())
		for i := 0; i < value.Len(); i++ {
			strs[i] = currentIndent + indent + strIndentValue(value.Index(i), indent, currentIndent+indent, newlineChar, quoteStr)
		}

		return "[" + newlineChar + strings.Join(strs, stringSep) + newlineChar + currentIndent + "]"

	case reflect.Map:
		if value.Len() == 0 {
			return "{}"
		}
		strs := make([]string, value.Len())
		for i, keyValue := range value.MapKeys() {
			keyStr := strIndentValue(keyValue, indent, currentIndent+indent, newlineChar, quoteStr)
			valStr := strIndentValue(value.MapIndex(keyValue), indent, currentIndent+indent, newlineChar, quoteStr)
			strs[i] = currentIndent + indent + keyStr + ": " + valStr
		}
		sort.Strings(strs)
		return "{" + newlineChar + strings.Join(strs, stringSep) + newlineChar + currentIndent + "}"

	case reflect.Struct:
		if value.NumField() == 0 {
			return "{}"
		}

		strs := make([]string, value.NumField())

		for i := 0; i < value.NumField(); i++ {
			structField := valueType.Field(i)
			keyStr := strIndent(structField.Name, indent, currentIndent+indent, newlineChar, quoteStr)
			if tag, ok := structField.Tag.Lookup("yaml"); ok {
				keyStr = strIndent(strings.Split(tag, ",")[0], indent, currentIndent+indent, newlineChar, quoteStr)
			}
			if tag, ok := structField.Tag.Lookup("json"); ok {
				keyStr = strIndent(strings.Split(tag, ",")[0], indent, currentIndent+indent, newlineChar, quoteStr)
			}
			valStr := strIndentValue(value.Field(i), indent, currentIndent+indent, newlineChar, quoteStr)
			strs[i] = currentIndent + indent + keyStr + ": " + valStr
		}
		return "{" + newlineChar + strings.Join(strs, stringSep) + newlineChar + currentIndent + "}"

	case reflect.Ptr:
		return strIndentValue(reflect.Indirect(value), indent, currentIndent, newlineChar, quoteStr)

	case reflect.UnsafePointer:
		return strIndentValue(reflect.Indirect(value), indent, currentIndent+indent, newlineChar, quoteStr)

	case reflect.Interface:
		return strIndentValue(value.Elem(), indent, currentIndent+indent, newlineChar, quoteStr)

	case reflect.Func:
		return "<function>"

	case reflect.Chan:
		return "<channel>"

	case reflect.Invalid:
		return "<invalid>"

	default:
		return fmt.Sprint(val)
	}
}

func strIndentValue(val reflect.Value, indent string, currentIndent string, newlineChar string, quoteStr string) string {
	if val.IsValid() && val.CanInterface() {
		return strIndent(val.Interface(), indent, currentIndent, newlineChar, quoteStr)
	}
	return "<hidden>"
}

func YesNo(val bool) string {
	if val {
		return "yes"
	}
	return "no"
}

func Obj(val interface{}) string {
	return strIndent(val, "  ", "", "\n", `"`)
}

// Same as Obj(), but trim leading and trailing quotes if it's just a string
func ObjStripped(val interface{}) string {
	return TrimPrefixAndSuffix(Obj(val), `"`)
}

func ObjFlat(val interface{}) string {
	return strIndent(val, "", "", "", `"`)
}

func ObjFlatNoQuotes(val interface{}) string {
	return strIndent(val, "", "", "", "")
}

func UserStr(val interface{}) string {
	return strIndent(val, "", "", "", `"`)
}

func UserStrValue(val reflect.Value) string {
	return strIndentValue(val, "", "", "", `"`)
}

func UserStrStripped(val interface{}) string {
	return TrimPrefixAndSuffix(UserStr(val), `"`)
}

func UserStrs(val interface{}) []string {
	if val == nil {
		return nil
	}

	if reflect.TypeOf(val).Kind() != reflect.Slice {
		val = []interface{}{val}
	}

	inVal := reflect.ValueOf(val)
	if inVal.IsNil() {
		return nil
	}

	// Handle case where caller passed in a nested slice
	if inVal.Len() == 1 {
		if inVal.Index(0).Kind() == reflect.Slice {
			inVal = inVal.Index(0)
		} else if inVal.Index(0).Kind() == reflect.Interface { // Handle case where input is e.g. []interface{[]string{"test"}}
			firstElementVal := reflect.ValueOf(inVal.Index(0).Interface())
			if firstElementVal.Kind() == reflect.Slice {
				inVal = firstElementVal
			}
		}
	}

	out := make([]string, inVal.Len())
	for i := 0; i < inVal.Len(); i++ {
		out[i] = UserStrValue(inVal.Index(i))
	}
	return out
}

func Index(index int) string {
	return fmt.Sprintf("index %d", index)
}

func Indent(str string, indent string) string {
	if str == "" {
		return indent
	}

	if str[len(str)-1:] == "\n" {
		out := ""
		for _, line := range strings.Split(str[:len(str)-1], "\n") {
			out += indent + line + "\n"
		}
		return out
	}

	out := ""
	for _, line := range strings.Split(strings.TrimRight(str, "\n"), "\n") {
		out += indent + line + "\n"
	}
	return out[:len(out)-1]
}

func TruncateEllipses(str string, maxLength int) string {
	ellipses := " ..."
	if len(str) > maxLength {
		str = str[:maxLength-len(ellipses)]
		str += ellipses
	}
	return str
}
