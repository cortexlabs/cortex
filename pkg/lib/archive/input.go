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

package archive

import (
	"github.com/cortexlabs/cortex/pkg/lib/files"
)

type Input struct {
	Files          []FileInput
	Bytes          []BytesInput
	Dirs           []DirInput
	FileLists      []FileListInput
	AddPrefix      string   // Gets added to every item
	EmptyFiles     []string // Empty files to be created
	AllowOverwrite bool     // Don't error if a file in the zip is overwritten
}

type FileInput struct {
	Source string
	Dest   string
}

type BytesInput struct {
	Content []byte
	Dest    string
}

type DirInput struct {
	Source             string
	Dest               string
	IgnoreFns          []files.IgnoreFn
	Flatten            bool
	RemovePrefix       string
	RemoveCommonPrefix bool
}

type FileListInput struct {
	Sources            []string
	Dest               string
	Flatten            bool
	RemovePrefix       string
	RemoveCommonPrefix bool
}
