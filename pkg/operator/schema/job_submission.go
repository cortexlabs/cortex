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

package schema

import (
	"encoding/json"

	"github.com/cortexlabs/cortex/pkg/types/spec"
)

type ItemList struct {
	Items     []json.RawMessage `json:"items"`
	BatchSize int               `json:"batch_size"`
}

type S3Lister struct {
	S3Paths  []string `json:"s3_paths"` // s3://<bucket_name>/key
	Includes []string `json:"includes"`
	Excludes []string `json:"excludes"`
}

type FilePathLister struct {
	S3Lister
	BatchSize int `json:"batch_size"`
}

type DelimitedFiles struct {
	S3Lister
	BatchSize int `json:"batch_size"`
}

type BatchJobSubmission struct {
	spec.RuntimeBatchJobConfig
	ItemList       *ItemList       `json:"item_list"`
	FilePathLister *FilePathLister `json:"file_path_lister"`
	DelimitedFiles *DelimitedFiles `json:"delimited_files"`
}

type TaskJobSubmission struct {
	spec.RuntimeTaskJobConfig
}
