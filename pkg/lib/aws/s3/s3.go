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

package s3

import (
	"strings"

	s "github.com/cortexlabs/cortex/pkg/api/strings"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
)

func IsValidS3aPath(s3aPath string) bool {
	if !strings.HasPrefix(s3aPath, "s3a://") {
		return false
	}
	parts := strings.Split(s3aPath[6:], "/")
	if len(parts) < 2 {
		return false
	}
	if parts[0] == "" || parts[1] == "" {
		return false
	}
	return true
}

func SplitS3aPath(s3aPath string) (string, string, error) {
	if !IsValidS3aPath(s3aPath) {
		return "", "", errors.New(s.ErrInvalidS3aPath(s3aPath))
	}
	fullPath := s3aPath[6:]
	slashIndex := strings.Index(fullPath, "/")
	bucket := fullPath[0:slashIndex]
	key := fullPath[slashIndex+1:]

	return bucket, key, nil
}
