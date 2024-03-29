/*
Copyright 2022 Cortex Labs, Inc.

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

package enqueuer

import (
	"fmt"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
)

const (
	ErrFailedToEnqueueMessages = "batchapi.failed_to_enqueue_messages"
	ErrMessageExceedsMaxSize   = "batchapi.message_exceeds_max_size"
)

func ErrorFailedToEnqueueMessages(message string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrFailedToEnqueueMessages,
		Message: message,
	})
}

func ErrorMessageExceedsMaxSize(messageSize int, messageLimit int) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrMessageExceedsMaxSize,
		Message: fmt.Sprintf("cannot enqueue message because its size of %d bytes exceeds the %d bytes limit; use a smaller batch size or reduce the size of each of item in the batch", messageSize, messageLimit),
	})
}
