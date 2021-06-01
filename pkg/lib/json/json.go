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

package json

import (
	"bytes"
	"encoding/json"
	"path/filepath"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
)

func MarshalIndent(obj interface{}) ([]byte, error) {
	jsonBytes, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return nil, errors.Wrap(err, errStrMarshalJSON)
	}
	return jsonBytes, nil
}

func Marshal(obj interface{}) ([]byte, error) {
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		return nil, errors.Wrap(err, errStrMarshalJSON)
	}
	return jsonBytes, nil
}

func Unmarshal(jsonBytes []byte, dst interface{}) error {
	if err := json.Unmarshal(jsonBytes, dst); err != nil {
		return errors.Wrap(err, errStrUnmarshalJSON)
	}
	return nil
}

func DecodeWithNumber(jsonBytes []byte, dst interface{}) error {
	d := json.NewDecoder(bytes.NewReader(jsonBytes))
	d.UseNumber()
	if err := d.Decode(&dst); err != nil {
		return errors.Wrap(err, errStrUnmarshalJSON)
	}

	return nil
}

func MarshalJSONStr(obj interface{}) (string, error) {
	jsonBytes, err := Marshal(obj)
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}

func WriteJSON(obj interface{}, outPath string) error {
	jsonBytes, err := Marshal(obj)
	if err != nil {
		return err
	}
	if err := files.CreateDir(filepath.Dir(outPath)); err != nil {
		return err
	}

	if err := files.WriteFile(jsonBytes, outPath); err != nil {
		return err
	}
	return nil
}

func Pretty(obj interface{}) (string, error) {
	b, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return "", errors.Wrap(err, errStrMarshalJSON)
	}

	return string(b), nil
}
