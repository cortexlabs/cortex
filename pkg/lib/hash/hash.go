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

package hash

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/cortexlabs/cortex/pkg/lib/files"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
)

// Bytes will trim to 63 characters because e.g. K8s labels must be < 64
func Bytes(bytes []byte) string {
	hash := sha256.New()
	hash.Write(bytes)
	str := hex.EncodeToString(hash.Sum(nil))
	return str[:63]
}

func String(str string) string {
	return Bytes([]byte(str))
}

func Strings(strs ...string) string {
	return String(strings.Join(strs, ","))
}

func Any(obj interface{}) string {
	return String(s.Obj(obj))
}

func File(path string) (string, error) {
	fileBytes, err := files.ReadFileBytes(path)
	if err != nil {
		return "", err
	}
	return Bytes(fileBytes), nil
}
