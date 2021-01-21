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

package k8s

import (
	"fmt"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
)

const (
	ErrLabelNotFound      = "k8s.label_not_found"
	ErrAnnotationNotFound = "k8s.annotation_not_found"
	ErrParseLabel         = "k8s.parse_label"
	ErrParseAnnotation    = "k8s.parse_annotation"
	ErrParseQuantity      = "k8s.parse_quantity"
)

func ErrorLabelNotFound(labelName string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrLabelNotFound,
		Message: fmt.Sprintf("label %s not found", s.UserStr(labelName)),
	})
}

func ErrorAnnotationNotFound(annotationName string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrAnnotationNotFound,
		Message: fmt.Sprintf("annotation %s not found", s.UserStr(annotationName)),
	})
}

func ErrorParseLabel(labelName string, labelVal string, desiredType string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrParseLabel,
		Message: fmt.Sprintf("unable to parse value %s from label %s as type %s", s.UserStr(labelVal), labelName, desiredType),
	})
}

func ErrorParseAnnotation(annotationName string, annotationVal string, desiredType string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrParseAnnotation,
		Message: fmt.Sprintf("unable to parse value %s from annotation %s as type %s", s.UserStr(annotationVal), annotationName, desiredType),
	})
}

func ErrorParseQuantity(qtyStr string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrParseQuantity,
		Message: fmt.Sprintf("%s: invalid kubernetes quantity, some valid examples are 1, 200m, 500Mi, 2G (see here for more information: https://docs.cortex.dev/v/%s/)", qtyStr, consts.CortexVersionMinor),
	})
}
