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

package flags

import (
	"fmt"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
)

const (
	ErrInvalidOutputType = "cli.invalid_output_type"
)

func ErrorInvalidOutputType(invalidOutputType string, validOutputTypes []string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidOutputType,
		Message: fmt.Sprintf("invalid value specified for flag --output %s, valid output values are %s", invalidOutputType, s.StrsAnd(validOutputTypes)),
	})
}
