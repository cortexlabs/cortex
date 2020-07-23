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
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
)

const (
	ErrUnknownAPIGatewayType     = "userconfig.unknown_api_gateway_type"
	ErrConflictingFields         = "userconfig.conflicting_fields"
	ErrBatchItemSizeExceedsLimit = "userconfig.batch_item_size_exceeds_limit"
	ErrSpecifyExactlyOneKey      = "userconfig.specify_exactly_one_key"
)

func ErrorUnknownAPIGatewayType() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrUnknownAPIGatewayType,
		Message: "unknown api gateway type",
	})
}

func ErrorConflictingFields(key string, keys ...string) error {
	allKeys := append(keys, key)
	return errors.WithStack(&errors.Error{
		Kind:    ErrConflictingFields,
		Message: fmt.Sprintf("please specify either the %s field (both not more than one at the same time)", s.StrsOr(allKeys)),
	})
}

func ErrorItemSizeExceedsLimit(index int, size int, limit int) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrBatchItemSizeExceedsLimit,
		Message: fmt.Sprintf("item %d has size %d bytes which exceeds the limit %d", index, size, limit),
	})
}

func ErrorSpecifyExactlyOneKey(key string, keys ...string) error {
	allKeys := append(keys, key)
	return errors.WithStack(&errors.Error{
		Kind:    ErrSpecifyExactlyOneKey,
		Message: fmt.Sprintf("specify exactly one of the following keys %s", s.StrsOr(allKeys)), // TODO add job specification documentation
	})
}
