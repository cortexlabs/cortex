/*
Copyright 2020 Cortex Labs, Inc.

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

package files

import (
	"fmt"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
)

type ErrorKind int

const (
	ErrUnknown ErrorKind = iota
	ErrCreateDir
	ErrDeleteDir
	ErrReadFormFile
	ErrCreateFile
	ErrReadDir
	ErrReadFile
	ErrFileAlreadyExists
	ErrUnexpected
	ErrFileDoesNotExist
	ErrDirDoesNotExist
	ErrNotAFile
	ErrNotADir
)

var _errorKinds = []string{
	"files.unknown",
	"files.create_dir",
	"files.delete_dir",
	"files.read_form_file",
	"files.create_file",
	"files.read_dir",
	"files.read_file",
	"files.file_already_exists",
	"files.unexpected",
	"files.file_does_not_exist",
	"files.dir_does_not_exist",
	"files.not_a_file",
	"files.not_a_dir",
}

var _ = [1]int{}[int(ErrNotADir)-(len(_errorKinds)-1)] // Ensure list length matches

func (t ErrorKind) String() string {
	return _errorKinds[t]
}

// MarshalText satisfies TextMarshaler
func (t ErrorKind) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// UnmarshalText satisfies TextUnmarshaler
func (t *ErrorKind) UnmarshalText(text []byte) error {
	enum := string(text)
	for i := 0; i < len(_errorKinds); i++ {
		if enum == _errorKinds[i] {
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

func ErrorCreateDir(path string) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrCreateDir,
		Message: fmt.Sprintf("%s: unable to create directory", path),
	})
}

func ErrorDeleteDir(path string) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrDeleteDir,
		Message: fmt.Sprintf("%s: unable to delete directory", path),
	})
}

func ErrorReadFormFile(fileName string) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrReadFormFile,
		Message: fmt.Sprintf("unable to read request form file %s", s.UserStr(fileName)),
	})
}

func ErrorCreateFile(path string) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrCreateFile,
		Message: fmt.Sprintf("%s: unable to create file", path),
	})
}

func ErrorReadDir(path string) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrReadDir,
		Message: fmt.Sprintf("%s: unable to read directory", path),
	})
}

func ErrorReadFile(path string) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrReadFile,
		Message: fmt.Sprintf("%s: unable to read file", path),
	})
}

func ErrorFileAlreadyExists(path string) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrFileAlreadyExists,
		Message: fmt.Sprintf("%s: file already exists", path),
	})
}

func ErrorUnexpected() error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrUnexpected,
		Message: "an unexpected error occurred",
	})
}

func ErrorFileDoesNotExist(path string) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrFileDoesNotExist,
		Message: fmt.Sprintf("%s: file does not exist", path),
	})
}

func ErrorDirDoesNotExist(path string) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrDirDoesNotExist,
		Message: fmt.Sprintf("%s: directory does not exist", path),
	})
}

func ErrorNotAFile(path string) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrNotAFile,
		Message: fmt.Sprintf("%s: no such file", path),
	})
}

func ErrorNotADir(path string) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrNotADir,
		Message: fmt.Sprintf("%s: not a directory path", path),
	})
}
