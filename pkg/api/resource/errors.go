package resource

import (
	"fmt"

	s "github.com/cortexlabs/cortex/pkg/api/strings"
)

type ErrorKind int

const (
	ErrUnknown ErrorKind = iota
	ErrUnknownKind
	ErrNotFound
	ErrInvalidType
)

var (
	errorKinds = []string{
		"err_unknown",
		"err_unknown_kind",
		"err_not_found",
		"err_invalid_type",
	}
)

func (t ErrorKind) String() string {
	return errorKinds[t]
}

// MarshalText satisfies TextMarshaler
func (t ErrorKind) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// UnmarshalText satisfies TextUnmarshaler
func (t *ErrorKind) UnmarshalText(text []byte) error {
	enum := string(text)
	for i := 0; i < len(errorKinds); i++ {
		if enum == errorKinds[i] {
			*t = ErrorKind(i)
			return nil
		}
	}

	*t = ErrUnknown
	return nil
}

// UnmarshalBinary satisfies BinaryUnmarshaler
// Needed for msgpack
func (t *ErrorKind) UnmarshalBinary(data []byte) error {
	return t.UnmarshalText(data)
}

// MarshalBinary satisfies BinaryMarshaler
func (t ErrorKind) MarshalBinary() ([]byte, error) {
	return []byte(t.String()), nil
}

type ResourceError struct {
	Kind ErrorKind

	message string
}

func ErrorNotFound(name string, resourceType Type) error {
	return ResourceError{
		Kind:    ErrNotFound,
		message: fmt.Sprintf("%s %s not found", resourceType, s.UserStr(name)),
	}
}

func ErrorInvalidType(invalid string) error {
	return ResourceError{
		Kind:    ErrInvalidType,
		message: fmt.Sprintf("invalid resource type %s", s.UserStr(invalid)),
	}
}

func ErrorUnknownKind(name string) error {
	return ResourceError{
		Kind:    ErrUnknownKind,
		message: fmt.Sprintf("unknown kind %s", s.UserStr(name)),
	}
}

func (e ResourceError) Error() string {
	return e.message
}
