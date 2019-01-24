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

package util

import (
	"crypto/sha256"
	"encoding/hex"
	"io/ioutil"

	s "github.com/cortexlabs/cortex/pkg/api/strings"
	"github.com/cortexlabs/cortex/pkg/utils/errors"
)

// Trim to 63 characters because e.g. K8s labels must be < 64
func HashBytes(bytes []byte) string {
	hash := sha256.New()
	hash.Write(bytes)
	str := hex.EncodeToString(hash.Sum(nil))
	return str[:63]
}

func HashStr(str string) string {
	return HashBytes([]byte(str))
}

func HashObj(obj interface{}) string {
	return HashStr(s.Obj(obj))
}

func HashFile(path string) (string, error) {
	fileBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return "", errors.Wrap(err, s.ErrReadFile(path))
	}
	return HashBytes(fileBytes), nil
}
