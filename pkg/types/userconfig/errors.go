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

package userconfig

import (
	"fmt"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
)

const (
	ErrUnknownAPIGatewayType     = "userconfig.unknown_api_gateway_type"
	ErrConflictingFields         = "userconfig.conflicting_fields"
	ErrBatchItemSizeExceedsLimit = "userconfig.batch_item_size_exceeds_limit"
)

func ErrorUnknownAPIGatewayType() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrUnknownAPIGatewayType,
		Message: "unknown api gateway type",
	})
}

func ErrorConflictingFields(fieldKeyA, fieldKeyB string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrConflictingFields,
		Message: fmt.Sprintf("please specify either the %s or %s field (both cannot be specified at the same time)", fieldKeyA, fieldKeyB),
	})
}

func ErrorItemSizeExceedsLimit(size int, limit int) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrBatchItemSizeExceedsLimit,
		Message: fmt.Sprintf("batch item has size %d bytes which exceeds the limit %d", size, limit),
	})
}
