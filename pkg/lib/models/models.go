/*
Copyright 2019 Cortex Labs, Inc.

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

package models

import (
	"fmt"

	"github.com/cortexlabs/cortex/pkg/lib/aws"
)

// IsValidS3Directory checks that the path contains a valid S3 directory for Tensorflow models
// Must contain the following structure:
// - 1523423423/ (version prefix, usually a timestamp)
// 		- saved_model.pb
//		- variables/
//			- variables.index
//			- variables.data-00000-of-00001 (there are a variable number of these files)
func IsValidS3Directory(path string) bool {
	if valid, err := aws.IsS3PathFileExternal(
		fmt.Sprintf("%s/saved_model.pb", path),
		fmt.Sprintf("%s/variables/variables.index", path),
	); err != nil || !valid {
		return false
	}

	if valid, err := aws.IsS3PathPrefixExternal(
		fmt.Sprintf("%s/variables/variables.data-00000-of", path),
	); err != nil || !valid {
		return false
	}
	return true
}
