package main

import "fmt"

type ErrorKind int

const (
	ErrUnknown ErrorKind = iota
	ErrAPIVersionMismatch
)

var (
	errorKinds = []string{
		"err_unknown",
		"err_api_version_mismatch",
	}
)

var _ = [1]int{}[int(ErrAPIVersionMismatch)-(len(errorKinds)-1)] // Ensure list length matches

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

type Error struct {
	Kind    ErrorKind
	message string
}

func (e Error) Error() string {
	return e.message
}

func ErrorAPIVersionMismatch(operatorVersion string, clientVersion string) error {
	return Error{
		Kind:    ErrAPIVersionMismatch,
		message: fmt.Sprintf("API version mismatch (Operator: %s; Client: %s)", operatorVersion, clientVersion),
	}
}
