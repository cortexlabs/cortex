package userconfig

import "strings"

type ValueType int
type ValueTypes []ValueType

const (
	UnknownValueType ValueType = iota
	IntegerValueType
	FloatValueType
	StringValueType
	BoolValueType
)

var valueTypes = []string{
	"unknown",
	"INT",
	"FLOAT",
	"STRING",
	"BOOL",
}

func ValueTypeFromString(s string) ValueType {
	for i := 0; i < len(valueTypes); i++ {
		if s == valueTypes[i] {
			return ValueType(i)
		}
	}
	return UnknownValueType
}

func ValueTypeStrings() []string {
	return valueTypes[1:]
}

func (t ValueType) String() string {
	return valueTypes[t]
}

// MarshalText satisfies TextMarshaler
func (t ValueType) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// UnmarshalText satisfies TextUnmarshaler
func (t *ValueType) UnmarshalText(text []byte) error {
	enum := string(text)
	for i := 0; i < len(valueTypes); i++ {
		if enum == valueTypes[i] {
			*t = ValueType(i)
			return nil
		}
	}

	*t = UnknownValueType
	return nil
}

// UnmarshalBinary satisfies BinaryUnmarshaler
// Needed for msgpack
func (t *ValueType) UnmarshalBinary(data []byte) error {
	return t.UnmarshalText(data)
}

// MarshalBinary satisfies BinaryMarshaler
func (t ValueType) MarshalBinary() ([]byte, error) {
	return []byte(t.String()), nil
}

func (ts ValueTypes) StringList() []string {
	strs := make([]string, len(ts))
	for i, t := range ts {
		strs[i] = t.String()
	}
	return strs
}

func (ts ValueTypes) String() string {
	return strings.Join(ts.StringList(), ", ")
}
