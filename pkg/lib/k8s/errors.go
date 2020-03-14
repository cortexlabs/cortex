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

package k8s

import (
	"fmt"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
)

type ErrorKind int

const (
	ErrUnknown            ErrorKind = iota
	ErrAnnotationNotFound ErrorKind = iota
	ErrParseAnnotation    ErrorKind = iota
	ErrParseQuantity      ErrorKind = iota
)

var _errorKinds = []string{
	"k8s.unknown",
	"k8s.annotation_not_found",
	"k8s.parse_annotation",
	"k8s.parse_quantity",
}

var _ = [1]int{}[int(ErrParseQuantity)-(len(_errorKinds)-1)] // Ensure list length matches

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

func ErrorAnnotationNotFound(annotationName string) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrAnnotationNotFound,
		Message: fmt.Sprintf("annotation %s not found", s.UserStr(annotationName)),
	})
}

func ErrorParseAnnotation(annotationName string, annotationVal string, desiredType string) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrParseAnnotation,
		Message: fmt.Sprintf("unable to parse value %s from annotation %s as type %s", s.UserStr(annotationVal), annotationName, desiredType),
	})
}

func ErrorParseQuantity(qtyStr string) error {
	return errors.WithStack(&errors.CortexError{
		Kind: ErrParseQuantity,
		// CORTEX_VERSION_MINOR
		Message: qtyStr + ": invalid kubernetes quantity, some valid examples are 1, 200m, 500Mi, 2G (see here for more information: https://cortex.dev/v/master/deployments/compute)",
	})
}
