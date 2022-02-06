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

package batchapi

import (
	"fmt"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
)

const (
	ErrNoS3FilesFound            = "batchapi.no_s3_files_found"
	ErrBatchItemSizeExceedsLimit = "batchapi.item_size_exceeds_limit"
)

func ErrorNoS3FilesFound() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrNoS3FilesFound,
		Message: "no s3 files match search criteria",
	})
}

func ErrorItemSizeExceedsLimit(index int, size int, limit int) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrBatchItemSizeExceedsLimit,
		Message: fmt.Sprintf("item %d has size %d bytes which exceeds the limit (%d bytes)", index, size, limit),
	})
}
