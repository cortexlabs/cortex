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

const (
	ErrCreateDir         = "files.create_dir"
	ErrDeleteDir         = "files.delete_dir"
	ErrReadFormFile      = "files.read_form_file"
	ErrCreateFile        = "files.create_file"
	ErrReadDir           = "files.read_dir"
	ErrReadFile          = "files.read_file"
	ErrFileAlreadyExists = "files.file_already_exists"
	ErrUnexpected        = "files.unexpected"
	ErrFileDoesNotExist  = "files.file_does_not_exist"
	ErrDirDoesNotExist   = "files.dir_does_not_exist"
	ErrNotAFile          = "files.not_a_file"
	ErrNotADir           = "files.not_a_dir"
)

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
