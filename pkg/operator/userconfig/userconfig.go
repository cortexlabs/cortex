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

package userconfig

import (
	"github.com/cortexlabs/cortex/pkg/lib/cast"
	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/zip"
)

// TODO only call in operator? move this to operator package?
func ValidateAPIs(apis []*API, projectBytes []byte) error {
	var err error
	projectFileMap := make(map[string][]byte)

	projectFileMap, err = zip.UnzipMemToMem(projectBytes)
	if err != nil {
		return err
	}

	dups := FindDuplicateNames(apis)
	if len(dups) > 0 {
		return ErrorDuplicateName(dups)
	}

	for _, api := range apis {
		if err := api.Validate(projectFileMap); err != nil {
			return err
		}
	}

	return nil
}

func FindDuplicateNames(apis []*API) []*API {
	names := make(map[string][]*API)

	for _, api := range apis {
		names[api.Name] = append(names[api.Name], api)
	}

	for name := range names {
		if len(names[name]) > 1 {
			return names[name]
		}
	}

	return nil
}

func New(filePath string, configBytes []byte) ([]*API, error) {
	var err error

	configData, err := cr.ReadYAMLBytes(configBytes)
	if err != nil {
		return nil, errors.Wrap(err, filePath)
	}

	configDataSlice, ok := cast.InterfaceToStrInterfaceMapSlice(configData)
	if !ok {
		return nil, errors.Wrap(ErrorMalformedConfig(), filePath)
	}

	apis := make([]*API, len(configDataSlice))
	for i, data := range configDataSlice {
		api := &API{}
		errs := cr.Struct(api, data, apiValidation)
		if errors.HasErrors(errs) {
			name, _ := data[NameKey].(string)
			return nil, errors.Wrap(errors.FirstError(errs...), identify(filePath, name, i))
		}

		api.Index = i
		api.FilePath = filePath
		apis = append(apis, api)
	}

	return apis, nil
}

func ReadConfigFile(filePath string, relativePath string) ([]*API, error) {
	configBytes, err := files.ReadFileBytesErrPath(filePath, relativePath)
	if err != nil {
		return nil, err
	}

	apis, err := New(relativePath, configBytes)
	if err != nil {
		return nil, err
	}

	return apis, nil
}
