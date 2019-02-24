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

package json

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	s "github.com/cortexlabs/cortex/pkg/api/strings"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
)

func MarshalJSON(obj interface{}) ([]byte, error) {
	jsonBytes, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return nil, errors.Wrap(err, s.ErrMarshalJSON)
	}
	return jsonBytes, nil
}

func MarshalJSONStr(obj interface{}) (string, error) {
	jsonBytes, err := MarshalJSON(obj)
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}

func WriteJSON(obj interface{}, outPath string) error {
	jsonBytes, err := MarshalJSON(obj)
	if err != nil {
		return err
	}
	err = os.MkdirAll(filepath.Dir(outPath), os.ModePerm)
	if err != nil {
		return errors.Wrap(err, s.ErrCreateDir(filepath.Dir(outPath)))
	}
	err = ioutil.WriteFile(outPath, jsonBytes, 0644)
	if err != nil {
		return errors.Wrap(err, s.ErrWriteFile(outPath))
	}
	return nil
}
