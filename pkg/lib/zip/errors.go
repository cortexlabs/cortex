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

package zip

import (
	"fmt"

	s "github.com/cortexlabs/cortex/pkg/api/strings"
)

type ErrorKind int

const (
	ErrUnknown ErrorKind = iota
	ErrDuplicateZipPath
	ErrCreateDir
	ErrCreateFile
)

var errorKinds = []string{
	"err_unknown",
	"err_duplicate_zip_path",
	"err_create_dir",
	"err_create_file",
}

var _ = [1]int{}[int(ErrCreateFile)-(len(errorKinds)-1)] // Ensure list length matches

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

func ErrorDuplicateZipPath(path string) error {
	return Error{
		Kind:    ErrDuplicateZipPath,
		message: fmt.Sprintf("conflicting path in zip (%s)", s.UserStr(path)),
	}
}

func ErrorCreateDir(path string) error {
	return Error{
		Kind:    ErrCreateDir,
		message: fmt.Sprintf("%s: unable to create directory", path),
	}
}

func ErrorCreateFile(path string) error {
	return Error{
		Kind:    ErrCreateFile,
		message: fmt.Sprintf("%s: unable to create file", path),
	}
}
